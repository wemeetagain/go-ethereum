package ssz

import (
	"encoding/binary"
	"fmt"
	"reflect"
)

type VectorType struct {
	ElementType SSZType
	Length      uint
}

func Vector(elementType SSZType, length uint) SSZType {
	return &VectorType{
		ElementType: elementType,
		Length:      length,
	}
}

func (t *VectorType) Name() string {
	return fmt.Sprintf("Vector[%s, %d]", t.ElementType.Name(), t.Length)
}

func (t *VectorType) Default() interface{} {
	return reflect.MakeSlice(reflect.SliceOf(t.ElementType.Type()), int(t.Length), int(t.Length)).Interface()
}

func (t *VectorType) Type() reflect.Type {
	return reflect.TypeOf(t.Default())
}

func (t *VectorType) IsVariableSize() bool {
	return t.ElementType.IsVariableSize()
}

func (t *VectorType) FixedSize() uint {
	return t.ElementType.FixedSize() * t.Length
}

func (t *VectorType) Size(v interface{}) (uint, error) {
	if !t.ElementType.IsVariableSize() {
		return t.ElementType.FixedSize() * t.Length, nil
	}

	vv := reflect.ValueOf(v)
	if vv.Kind() != reflect.Slice {
		return 0, fmt.Errorf("%s: type mismatch: %v", t.Name(), reflect.TypeOf(v))
	}
	l := vv.Len()
	if (uint)(l) != t.Length {
		return 0, fmt.Errorf("%s: length mismatch: %d != %d", t.Name(), l, t.Length)
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
	return size + 4*t.Length, nil
}

func (t *VectorType) HashTreeRoot(v interface{}) ([32]byte, error) {
	if isBasicType(t.ElementType) {
		s, err := t.Serialize(v)
		if err != nil {
			return [32]byte{}, err
		}
		chunks := Pack(s)
		chunkCount := (t.Length*t.ElementType.FixedSize() + 31) / 32
		return Merkleize(chunks, uint(chunkCount)), nil
	} else {
		var chunks [][32]byte
		vv := reflect.ValueOf(v)
		if vv.Kind() != reflect.Slice {
			return [32]byte{}, fmt.Errorf("%s: type mismatch: %v", t.Name(), reflect.TypeOf(v))
		}
		l := vv.Len()
		if (uint)(l) != t.Length {
			return [32]byte{}, fmt.Errorf("%s: length mismatch: %d != %d", t.Name(), l, t.Length)
		}
		for i := 0; i < l; i++ {
			e := vv.Index(i).Interface()
			r, err := t.ElementType.HashTreeRoot(e)
			if err != nil {
				return [32]byte{}, err
			}
			chunks = append(chunks, r)
		}
		return Merkleize(chunks, uint(t.Length)), nil
	}
}

func (t *VectorType) SerializeTo(v interface{}, b []byte, start int) error {
	vv := reflect.ValueOf(v)
	if vv.Kind() != reflect.Slice {
		return fmt.Errorf("%s: type mismatch: %v", t.Name(), reflect.TypeOf(v))
	}
	if vv.Len() != int(t.Length) {
		return fmt.Errorf("%s: length mismatch: %d", t.Name(), vv.Len())
	}

	if !t.ElementType.IsVariableSize() {
		for i := 0; i < vv.Len(); i++ {
			e := vv.Index(i).Interface()
			err := t.ElementType.SerializeTo(e, b, start+i*int(t.ElementType.FixedSize()))
			if err != nil {
				return err
			}
		}
		return nil
	} else {
		offset := 4 * t.Length
		for i := 0; i < vv.Len(); i++ {
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

func (t *VectorType) Serialize(v interface{}) ([]byte, error) {
	s, err := t.Size(v)
	if err != nil {
		return []byte{}, err
	}
	b := make([]byte, s)
	err = t.SerializeTo(v, b, 0)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (t *VectorType) DeserializeFrom(b []byte, start, end int) (interface{}, error) {
	if !t.ElementType.IsVariableSize() {
		fixedSize := t.ElementType.FixedSize()
		if (end - start) != int(fixedSize*t.Length) {
			return nil, fmt.Errorf("%s: length mismatch: %d != %d", t.Name(), end-start, fixedSize*t.Length)
		}
		v := reflect.ValueOf(t.Default())
		for i := uint(0); i < t.Length; i++ {
			e, err := t.ElementType.DeserializeFrom(b, start+int(i*fixedSize), start+int((i+1)*fixedSize))
			if err != nil {
				return nil, err
			}
			v.Index(int(i)).Set(reflect.ValueOf(e))
		}
		return v.Interface(), nil
	} else {
		firstOffset := binary.LittleEndian.Uint32(b[start:])
		if firstOffset != uint32(4*t.Length) {
			return nil, fmt.Errorf("%s: invalid offset: first offset must be == t.length", t.Name())
		}
		if firstOffset%4 != 0 {
			return nil, fmt.Errorf("%s: invalid offset: first offset must be a multiple of 4", t.Name())
		}
		offsets := []uint32{firstOffset}
		v := reflect.ValueOf(t.Default())
		for i := 1; i < int(t.Length); i++ {
			offset := binary.LittleEndian.Uint32(b[start+i*4:])
			if offset <= offsets[i-1] {
				return nil, fmt.Errorf("%s: invalid offset: offsets must be strictly increasing", t.Name())
			}
			offsets = append(offsets, offset)

			e, err := t.ElementType.DeserializeFrom(b, start+int(offsets[i-1]), start+int(offsets[i]))
			if err != nil {
				return nil, err
			}
			v.Index(i - 1).Set(reflect.ValueOf(e))
		}
		e, err := t.ElementType.DeserializeFrom(b, start+int(offsets[t.Length-1]), end)
		if err != nil {
			return nil, err
		}
		v.Index(int(t.Length) - 1).Set(reflect.ValueOf(e))

		return v.Interface(), nil
	}
}

func (t *VectorType) Deserialize(b []byte) (interface{}, error) {
	return t.DeserializeFrom(b, 0, len(b))
}
