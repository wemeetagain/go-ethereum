package ssz

import (
	"fmt"
	"reflect"
)

type BoolType struct{}

func Bool() SSZType {
	return &BoolType{}
}

func (t *BoolType) Name() string {
	return "bool"
}

func (t *BoolType) Type() reflect.Type {
	return reflect.TypeOf(t.Default())
}

func (t *BoolType) Default() interface{} {
	return false
}

func (t *BoolType) IsVariableSize() bool {
	return false
}

func (t *BoolType) FixedSize() uint {
	return 1
}

func (t *BoolType) Size(v interface{}) (uint, error) {
	return 1, nil
}

func (t *BoolType) HashTreeRoot(v interface{}) ([32]byte, error) {
	var b [32]byte
	err := t.SerializeTo(v, b[:], 0)
	if err != nil {
		return b, err
	}
	return b, nil
}

func (t *BoolType) SerializeTo(v interface{}, b []byte, start int) error {
	v_, ok := v.(bool)
	if !ok {
		return fmt.Errorf("bool: type mismatch: %v", reflect.TypeOf(v))
	}
	if v_ {
		b[start] = 1
	} else {
		b[start] = 0
	}
	return nil
}

func (t *BoolType) Serialize(v interface{}) ([]byte, error) {
	b := make([]byte, t.FixedSize())
	err := t.SerializeTo(v, b, 0)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (t *BoolType) DeserializeFrom(b []byte, start, end int) (interface{}, error) {
	if end-start != 1 {
		return nil, fmt.Errorf("bool: length mismatch: %d != 1", end-start)
	}
	if b[start] == 0 {
		return false, nil
	}
	if b[start] == 1 {
		return true, nil
	}
	return nil, fmt.Errorf("bool: invalid value: %d", b[start])
}

func (t *BoolType) Deserialize(b []byte) (interface{}, error) {
	return t.DeserializeFrom(b, 0, len(b))
}
