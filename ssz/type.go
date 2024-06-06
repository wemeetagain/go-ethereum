package ssz

import "reflect"

type SSZType interface {
	Name() string

	Default() interface{}
	Type() reflect.Type

	IsVariableSize() bool
	FixedSize() uint
	Size(v interface{}) (uint, error)

	HashTreeRoot(interface{}) ([32]byte, error)

	SerializeTo(interface{}, []byte, int) error
	Serialize(interface{}) ([]byte, error)

	DeserializeFrom([]byte, int, int) (interface{}, error)
	Deserialize([]byte) (interface{}, error)
}

func Equals(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}

func Serialize(t SSZType, v interface{}) ([]byte, error) {
	return t.Serialize(v)
}

func Deserialize(t SSZType, b []byte) (interface{}, error) {
	return t.Deserialize(b)
}
