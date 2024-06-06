package ssz

import (
	"fmt"
	"reflect"
)

type ByteVectorType struct {
	Length uint
}

func ByteVector(length uint) SSZType {
	return &ByteVectorType{
		Length: length,
	}
}

func (t *ByteVectorType) Name() string {
	return fmt.Sprintf("ByteVector[%d]", t.Length)
}

func (t *ByteVectorType) Default() interface{} {
	return make([]byte, t.Length)
}

func (t *ByteVectorType) Type() reflect.Type {
	return reflect.TypeOf(t.Default())
}

func (t *ByteVectorType) IsVariableSize() bool {
	return false
}

func (t *ByteVectorType) FixedSize() uint {
	return t.Length
}

func (t *ByteVectorType) Size(v interface{}) (uint, error) {
	return t.Length, nil
}

func (t *ByteVectorType) HashTreeRoot(v interface{}) ([32]byte, error) {
	s, err := t.Serialize(v)
	if err != nil {
		return [32]byte{}, err
	}
	chunks := Pack(s)
	return Merkleize(chunks, uint(len(chunks))), nil
}

func (t *ByteVectorType) SerializeTo(v interface{}, b []byte, start int) error {
	vv, ok := v.([]byte)
	if !ok {
		return fmt.Errorf("%s: type mismatch: %v", t.Name(), reflect.TypeOf(v))
	}
	if (uint)(len(vv)) != t.Length {
		return fmt.Errorf("%s: length does not match: %d != %d", t.Name(), len(vv), t.Length)
	}
	copy(b[start:], vv)
	return nil
}

func (t *ByteVectorType) Serialize(v interface{}) ([]byte, error) {
	b := make([]byte, t.FixedSize())
	err := t.SerializeTo(v, b, 0)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (t *ByteVectorType) DeserializeFrom(b []byte, start, end int) (interface{}, error) {
	if end-start != int(t.Length) {
		return nil, fmt.Errorf("%s: length mismatch: %d != %d", t.Name(), end-start, t.Length)
	}
	v := make([]byte, t.Length)
	n := copy(v, b[start:end])
	if n != int(t.Length) {
		return nil, fmt.Errorf("%s: copy length mismatch: %d != %d", t.Name(), n, t.Length)
	}
	return v, nil
}

func (t *ByteVectorType) Deserialize(b []byte) (interface{}, error) {
	return t.DeserializeFrom(b, 0, len(b))
}
