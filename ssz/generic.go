package ssz

import (
	"fmt"
	"reflect"
)

type GenSSZType[T any] struct {
	T SSZType
}

func NewGenSSZType[T any](t SSZType) (g GenSSZType[T], err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Invalid type: %v", r)
		}
	}()

	g = GenSSZType[T]{T: t}
	g.Default()
	return g, nil

}

func (g *GenSSZType[T]) Name() string {
	return g.T.Name()
}

func (g *GenSSZType[T]) Default() T {
	return g.T.Default().(T)
}

func (g *GenSSZType[T]) Type() reflect.Type {
	return g.T.Type()
}

func (g *GenSSZType[T]) IsVariableSize() bool {
	return g.T.IsVariableSize()
}

func (g *GenSSZType[T]) FixedSize() uint {
	return g.T.FixedSize()
}

func (g *GenSSZType[T]) Size(v T) (uint, error) {
	return g.T.Size(v)
}

func (g *GenSSZType[T]) HashTreeRoot(v T) ([32]byte, error) {
	return g.T.HashTreeRoot(v)
}

func (g *GenSSZType[T]) SerializeTo(v T, b []byte, i int) error {
	return g.T.SerializeTo(v, b, i)
}

func (g *GenSSZType[T]) Serialize(v T) ([]byte, error) {
	return g.T.Serialize(v)
}

func (g *GenSSZType[T]) DeserializeFrom(b []byte, i int, j int) (T, error) {
	var vv T
	v, err := g.T.DeserializeFrom(b, i, j)
	if err != nil {
		return vv, err
	}
	vv, ok := v.(T)
	if !ok {
		return vv, fmt.Errorf("%s: deserialized type could not be type asserted: %v", g.Name(), reflect.TypeOf(v))
	}
	return v.(T), nil
}

func (g *GenSSZType[T]) Deserialize(b []byte) (T, error) {
	return g.DeserializeFrom(b, 0, len(b))
}

func (g *GenSSZType[T]) Equals(a, b T) bool {
	return reflect.DeepEqual(a, b)
}
