package ssz

import (
	"encoding/binary"
	"fmt"
	"reflect"
)

type ContainerType struct {
	FieldNames       []string
	FieldTypes       []SSZType
	typ              reflect.Type
	variableOffsets  []int
	fixedFieldRanges [][2]int
	fixedEnd         int
}

type Field struct {
	K string
	V SSZType
}

func Container(fields []Field, typ interface{}) SSZType {
	fieldNames := make([]string, len(fields))
	fieldTypes := make([]SSZType, len(fields))
	t := reflect.TypeOf(typ)
	for i, f := range fields {
		fieldNames[i] = f.K
		fieldTypes[i] = f.V
	}
	fixedFieldRanges, variableOffsets, fixedEnd := preCalculateFieldRanges(fieldTypes)
	return &ContainerType{
		FieldNames:       fieldNames,
		FieldTypes:       fieldTypes,
		typ:              t,
		fixedFieldRanges: fixedFieldRanges,
		variableOffsets:  variableOffsets,
		fixedEnd:         fixedEnd,
	}
}

func (t *ContainerType) Name() string {
	return fmt.Sprintf("Container[%s]", t.typ.Name())
}

func (t *ContainerType) Default() interface{} {
	v := reflect.New(t.typ)
	fieldValues := make([]interface{}, len(t.FieldNames))
	for i, ft := range t.FieldTypes {
		fieldValues[i] = ft.Default()
	}
	vv, _ := t.SetFieldValues(v, fieldValues)
	return vv
}

func (t *ContainerType) Type() reflect.Type {
	return t.typ
}

func (t *ContainerType) IsVariableSize() bool {
	for _, ft := range t.FieldTypes {
		if ft.IsVariableSize() {
			return true
		}
	}
	return false
}

func (t *ContainerType) FixedSize() uint {
	var size uint
	for _, ft := range t.FieldTypes {
		if ft.IsVariableSize() {
			return 0
		}
		size += ft.FixedSize()
	}
	return size
}

func (t *ContainerType) Size(v interface{}) (uint, error) {
	fieldValues, err := t.GetFieldValues(v)
	if err != nil {
		return 0, err
	}
	var size uint
	for i, e := range fieldValues {
		if !t.FieldTypes[i].IsVariableSize() {
			size += t.FieldTypes[i].FixedSize()
		} else {
			s, err := t.FieldTypes[i].Size(e)
			if err != nil {
				return 0, err
			}
			size += 4 + s
		}
	}
	return size, nil
}

func (t *ContainerType) HashTreeRoot(v interface{}) ([32]byte, error) {
	fieldValues, err := t.GetFieldValues(v)
	if err != nil {
		return [32]byte{}, err
	}
	var chunks [][32]byte
	for i, ft := range t.FieldTypes {
		chunk, err := ft.HashTreeRoot(fieldValues[i])
		if err != nil {
			return [32]byte{}, err
		}
		chunks = append(chunks, chunk)
	}
	return Merkleize(chunks, uint(len(chunks))), nil
}

func (t *ContainerType) SerializeTo(v interface{}, b []byte, start int) error {
	var err error
	fieldValues, err := t.GetFieldValues(v)
	if err != nil {
		return err
	}
	variableIx := 0
	fixedIx := 0

	variableIndex := start + t.fixedEnd
	for i := 0; i < len(t.FieldNames); i++ {
		fieldType := t.FieldTypes[i]
		fieldValue := fieldValues[i]
		if !fieldType.IsVariableSize() {
			err = t.FieldTypes[i].SerializeTo(fieldValue, b, start+int(t.fixedFieldRanges[fixedIx][0]))
			if err != nil {
				return err
			}
			fixedIx++
		} else {
			binary.LittleEndian.PutUint32(b[start+int(t.variableOffsets[variableIx]):], uint32(variableIndex-start))
			s, err := t.FieldTypes[i].Size(fieldValue)
			if err != nil {
				return err
			}
			err = t.FieldTypes[i].SerializeTo(fieldValue, b, variableIndex)
			variableIndex += int(s)
			if err != nil {
				return err
			}
			variableIx++
		}
	}
	return nil
}

func (t *ContainerType) Serialize(v interface{}) ([]byte, error) {
	s, err := t.Size(v)
	if err != nil {
		return nil, err
	}
	b := make([]byte, s)
	err = t.SerializeTo(v, b, 0)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (t *ContainerType) DeserializeFrom(b []byte, start, end int) (interface{}, error) {
	v := reflect.New(t.typ).Interface()
	if !t.IsVariableSize() {
		if end-start != int(t.fixedEnd) {
			return nil, fmt.Errorf("%s: length mismatch", t.Name())
		}
	}

	offsets, err := readOffsets(t, b, start, end)
	if err != nil {
		return nil, err
	}

	variableIx := 0
	fixedIx := 0
	fieldValues := make([]interface{}, len(t.FieldNames))
	for i := 0; i < len(t.FieldNames); i++ {
		var r [2]int
		if !t.FieldTypes[i].IsVariableSize() {
			r = t.fixedFieldRanges[fixedIx]
			fixedIx++
		} else {
			r = [2]int{offsets[variableIx], offsets[variableIx+1]}
			variableIx++
		}
		fieldValue, err := t.FieldTypes[i].DeserializeFrom(b, start+r[0], start+r[1])
		if err != nil {
			return nil, err
		}
		fieldValues[i] = fieldValue
	}
	v, err = t.SetFieldValues(v, fieldValues)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (t *ContainerType) Deserialize(b []byte) (interface{}, error) {
	return t.DeserializeFrom(b, 0, len(b))
}

func (t *ContainerType) GetFieldValues(v interface{}) ([]interface{}, error) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%s: expected a struct but got %s", t.Name(), val.Kind())
	}
	if val.Type() != t.Type() {
		return nil, fmt.Errorf("%s: expected %s but got %s", t.Name(), t.Type(), val.Type())
	}
	fields := make([]interface{}, len(t.FieldNames))
	for i, fieldName := range t.FieldNames {
		field := val.FieldByName(fieldName)
		if !field.IsValid() {
			return nil, fmt.Errorf("%s: field not found: %s", t.Name(), fieldName)
		}
		fields[i] = field.Interface()
	}
	return fields, nil
}

func (t *ContainerType) SetFieldValues(v interface{}, fieldValues []interface{}) (interface{}, error) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%s: expected a struct but got %s", t.Name(), val.Kind())
	}
	if val.Type() != t.Type() {
		return nil, fmt.Errorf("%s: expected %s but got %s", t.Name(), t.Type(), val.Type())
	}
	for i, fieldName := range t.FieldNames {
		field := val.FieldByName(fieldName)
		if !field.IsValid() {
			return nil, fmt.Errorf("%s: field not found: %s", t.Name(), fieldName)
		}
		value := reflect.ValueOf(fieldValues[i])
		if value.Kind() == reflect.Ptr && field.Kind() != reflect.Ptr {
			value = value.Elem()
		}
		if field.Kind() != value.Kind() {
			return nil, fmt.Errorf("%s: expected %s but got %s", t.Name(), field.Kind(), value.Kind())
		}
		field.Set(value)
	}
	return val.Interface(), nil
}

func readOffsets(t *ContainerType, b []byte, start, end int) ([]int, error) {
	size := end - start
	offsets := make([]int, len(t.variableOffsets))
	for i, o := range t.variableOffsets {
		offset := int(binary.LittleEndian.Uint32(b[start+o:]))
		if offset > size {
			return nil, fmt.Errorf("%s: offset out of bounds: %d > %d", t.Name(), offset, size)
		}
		if i == 0 {
			if offset != t.fixedEnd {
				return nil, fmt.Errorf("%s: first offset must equal to fixedEnd: %d != %d", t.Name(), offset, t.fixedEnd)
			}
		} else {
			if offset < offsets[i-1] {
				return nil, fmt.Errorf("%s: offsets must be increasing: %d < %d", t.Name(), offset, offsets[i-1])
			}
		}
		offsets[i] = offset
	}
	offsets = append(offsets, size)
	return offsets, nil
}

func preCalculateFieldRanges(fieldTypes []SSZType) ([][2]int, []int, int) {
	var fixedFieldRanges [][2]int
	var variableOffsets []int
	var offset int
	for _, ft := range fieldTypes {
		if !ft.IsVariableSize() {
			s := int(ft.FixedSize())
			fixedFieldRanges = append(fixedFieldRanges, [2]int{offset, offset + s})
			offset += s
		} else {
			variableOffsets = append(variableOffsets, offset)
			offset += 4
		}
	}
	return fixedFieldRanges, variableOffsets, offset
}
