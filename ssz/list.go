package ssz

import (
	"encoding/binary"
	"fmt"
	"reflect"
)

type ListType struct {
	ElementType SSZType
	Limit       uint
}

func List(elementType SSZType, limit uint) SSZType {
	return &ListType{
		ElementType: elementType,
		Limit:       limit,
	}
}

func (t *ListType) Name() string {
	return fmt.Sprintf("List[%s, %d]", t.ElementType.Name(), t.Limit)
}

func (t *ListType) Default() interface{} {
	return reflect.MakeSlice(reflect.SliceOf(t.ElementType.Type()), 0, 0).Interface()
}

func (t *ListType) Type() reflect.Type {
	return reflect.TypeOf(t.Default())
}

func (t *ListType) IsVariableSize() bool {
	return true
}

func (t *ListType) FixedSize() uint {
	return 0
}

func (t *ListType) Size(v interface{}) (uint, error) {
	vv := reflect.ValueOf(v)
	if vv.Kind() != reflect.Slice {
		return 0, fmt.Errorf("%s: type mismatch: %v", t.Name(), reflect.TypeOf(v))
	}
	l := vv.Len()
	if (uint)(l) > t.Limit {
		return 0, fmt.Errorf("%s: length exceeds limit: %d > %d", t.Name(), l, t.Limit)
	}
	if isBasicType(t.ElementType) {
		return t.ElementType.FixedSize() * uint(l), nil
	}
	var size uint
	for i := 0; i < l; i++ {
		e := vv.Index(i).Interface()
		s, err := t.ElementType.Size(e)
		if err != nil {
			return 0, err
		}
		size += s
	}
	return size + 4*uint(l), nil
}

func (t *ListType) HashTreeRoot(v interface{}) ([32]byte, error) {
	vv := reflect.ValueOf(v)
	if vv.Kind() != reflect.Slice {
		return [32]byte{}, fmt.Errorf("%s: type mismatch: %v", t.Name(), reflect.TypeOf(v))
	}
	l := vv.Len()
	if (uint)(l) > t.Limit {
		return [32]byte{}, fmt.Errorf("%s: length exceeds limit: %d > %d", t.Name(), l, t.Limit)
	}
	if isBasicType(t.ElementType) {
		s, err := t.Serialize(v)
		if err != nil {
			return [32]byte{}, err
		}
		chunks := Pack(s)
		chunkCount := (t.Limit*t.ElementType.FixedSize() + 31) / 32
		return MixInLength(Merkleize(chunks, uint(chunkCount)), uint(l)), nil
	}
	chunks := make([][32]byte, l)
	for i := 0; i < l; i++ {
		e := vv.Index(i).Interface()
		h, err := t.ElementType.HashTreeRoot(e)
		if err != nil {
			return [32]byte{}, err
		}
		chunks[i] = h
	}
	return MixInLength(Merkleize(chunks, uint(t.Limit)), uint(l)), nil
}

func (t *ListType) SerializeTo(v interface{}, b []byte, start int) error {
	vv := reflect.ValueOf(v)
	if vv.Kind() != reflect.Slice {
		return fmt.Errorf("%s: type mismatch: %v", t.Name(), reflect.TypeOf(v))
	}
	l := vv.Len()
	if (uint)(l) > t.Limit {
		return fmt.Errorf("%s: length exceeds limit: %d > %d", t.Name(), l, t.Limit)
	}
	if !t.ElementType.IsVariableSize() {
		for i := 0; i < l; i++ {
			e := vv.Index(i).Interface()
			err := t.ElementType.SerializeTo(e, b, start+i*int(t.ElementType.FixedSize()))
			if err != nil {
				return err
			}
		}
		return nil
	} else {
		offset := 4 * uint(l)
		for i := 0; i < l; i++ {
			e := vv.Index(i).Interface()
			binary.LittleEndian.PutUint32(b[start+i*4:], uint32(offset))
			err := t.ElementType.SerializeTo(e, b, start+int(offset))
			s, err := t.ElementType.Size(e)
			if err != nil {
				return err
			}
			offset += s
		}
		return nil
	}
}

func (t *ListType) Serialize(v interface{}) ([]byte, error) {
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

func (t *ListType) DeserializeFrom(b []byte, start, end int) (interface{}, error) {
	if !t.ElementType.IsVariableSize() {
		fixedSize := int(t.ElementType.FixedSize())
		l := (end - start) / fixedSize
		if (end-start)%fixedSize != 0 {
			return nil, fmt.Errorf("%s: length mismatch, not aligned to element size", t.Name())
		}
		if l > int(t.Limit) {
			return nil, fmt.Errorf("%s: length exceeds limit: %d > %d", t.Name(), l, t.Limit)
		}
		v := reflect.MakeSlice(reflect.SliceOf(t.ElementType.Type()), l, l)
		for i := 0; i < int(l); i++ {
			e, err := t.ElementType.DeserializeFrom(b, start+i*fixedSize, start+(i+1)*fixedSize)
			if err != nil {
				return nil, err
			}

			v.Index(i).Set(reflect.ValueOf(e))
		}
		return v.Interface(), nil
	} else {
		firstOffset := binary.LittleEndian.Uint32(b[start:])
		if firstOffset%4 != 0 {
			return nil, fmt.Errorf("%s: invalid offset: first offset must be a multiple of 4", t.Name())
		}
		l := uint(firstOffset / 4)
		if l > t.Limit {
			return nil, fmt.Errorf("%s: length exceeds limit", t.Name())
		}
		v := reflect.MakeSlice(reflect.SliceOf(t.ElementType.Type()), int(l), int(l))
		offsets := []uint32{firstOffset}
		for i := 1; i < int(l); i++ {
			offset := binary.LittleEndian.Uint32(b[start+i*4:])
			if offset <= offsets[i-1] {
				return nil, fmt.Errorf("%s: invalid offset: offsets must be strictly increasing", t.Name())
			}
			offsets = append(offsets, offset)

			e, err := t.ElementType.DeserializeFrom(b, start+int(offsets[i-1]), start+int(offsets[i]))
			if err != nil {
				return nil, err
			}
			v.Index(i).Set(reflect.ValueOf(e))
		}
		e, err := t.ElementType.DeserializeFrom(b, start+int(offsets[l-1]), end)
		if err != nil {
			return nil, err
		}
		v.Index(int(l) - 1).Set(reflect.ValueOf(e))

		return v.Interface(), nil
	}
}

func (t *ListType) Deserialize(b []byte) (interface{}, error) {
	return t.DeserializeFrom(b, 0, len(b))
}
