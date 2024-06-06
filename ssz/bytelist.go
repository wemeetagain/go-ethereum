package ssz

import (
	"fmt"
	"reflect"
)

type ByteListType struct {
	Limit uint
}

func ByteList(limit uint) SSZType {
	return &ByteListType{
		Limit: limit,
	}
}

func (t *ByteListType) Name() string {
	return fmt.Sprintf("ByteVector[%d]", t.Limit)
}

func (t *ByteListType) Default() interface{} {
	return make([]byte, 0)
}

func (t *ByteListType) Type() reflect.Type {
	return reflect.TypeOf(t.Default())
}

func (t *ByteListType) IsVariableSize() bool {
	return true
}

func (t *ByteListType) FixedSize() uint {
	return 0
}

func (t *ByteListType) Size(v interface{}) (uint, error) {
	vv, ok := v.([]byte)
	if !ok {
		return 0, fmt.Errorf("%s: type mismatch: %v", t.Name(), reflect.TypeOf(v))
	}
	return uint(len(vv)), nil
}

func (t *ByteListType) HashTreeRoot(v interface{}) ([32]byte, error) {
	s, err := t.Serialize(v)
	if err != nil {
		return [32]byte{}, err
	}
	chunks := Pack(s)
	chunkCount := (t.Limit + 31) / 32
	return MixInLength(Merkleize(chunks, uint(chunkCount)), uint(len(v.([]byte)))), nil
}

func (t *ByteListType) SerializeTo(v interface{}, b []byte, start int) error {
	vv, ok := v.([]byte)
	if !ok {
		return fmt.Errorf("%s: type mismatch: %v", t.Name(), reflect.TypeOf(v))
	}
	if (uint)(len(vv)) > t.Limit {
		return fmt.Errorf("%s: length exceeds limit: %d > %d", t.Name(), len(vv), t.Limit)
	}
	copy(b[start:], vv)
	return nil
}

func (t *ByteListType) Serialize(v interface{}) ([]byte, error) {
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

func (t *ByteListType) DeserializeFrom(b []byte, start, end int) (interface{}, error) {
	l := end - start
	if (uint)(l) > t.Limit {
		return nil, fmt.Errorf("%s: length exceeds limit: %d > %d", t.Name(), l, t.Limit)
	}
	v := make([]byte, l)
	copy(v, b[start:end])
	return v, nil
}

func (t *ByteListType) Deserialize(b []byte) (interface{}, error) {
	return t.DeserializeFrom(b, 0, len(b))
}
