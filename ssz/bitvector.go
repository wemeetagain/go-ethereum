package ssz

import (
	"fmt"
	"math/bits"
	"reflect"
)

type BitVectorType struct {
	Length uint
}

func BitVector(length uint) SSZType {
	return &BitVectorType{
		Length: length,
	}
}

func (t *BitVectorType) Name() string {
	return fmt.Sprintf("BitVector[%d]", t.Length)
}

func (t *BitVectorType) Default() interface{} {
	return BitArray{
		Data:   make([]byte, (t.Length+7)/8),
		Bitlen: t.Length,
	}
}

func (t *BitVectorType) Type() reflect.Type {
	return reflect.TypeOf(t.Default())
}

func (t *BitVectorType) IsVariableSize() bool {
	return false
}

func (t *BitVectorType) FixedSize() uint {
	return (t.Length + 7) / 8
}

func (t *BitVectorType) Size(v interface{}) (uint, error) {
	switch v := v.(type) {
	case *BitArray:
		_ = v
	case BitArray:
		_ = v
	default:
		return 0, fmt.Errorf("%s: type mismatch: %v", t.Name(), reflect.TypeOf(v))
	}
	return t.FixedSize(), nil
}

func (t *BitVectorType) HashTreeRoot(v interface{}) ([32]byte, error) {
	var vv BitArray
	switch v := v.(type) {
	case *BitArray:
		vv = *v
	case BitArray:
		vv = v
	default:
		return [32]byte{}, fmt.Errorf("%s: type mismatch: %v", t.Name(), reflect.TypeOf(v))
	}
	chunks := Pack(vv.Data)
	return Merkleize(chunks, uint((t.Length+255)/256)), nil
}

func (t *BitVectorType) SerializeTo(v interface{}, b []byte, start int) error {
	var vv BitArray
	switch v := v.(type) {
	case *BitArray:
		vv = *v
	case BitArray:
		vv = v
	default:
		return fmt.Errorf("%s: type mismatch: %v", t.Name(), reflect.TypeOf(v))
	}
	if vv.Bitlen != t.Length {
		return fmt.Errorf("%s: length does not match: %d != %d", t.Name(), vv.Bitlen, t.Length)
	}
	s := (int(t.Length) + 7) / 8
	copy(b[start:start+s], vv.Data[:])
	return nil
}

func (t *BitVectorType) Serialize(v interface{}) ([]byte, error) {
	size, err := t.Size(v)
	if err != nil {
		return nil, err
	}
	b := make([]byte, size)
	err = t.SerializeTo(v, b, 0)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (t *BitVectorType) DeserializeFrom(b []byte, start, end int) (interface{}, error) {
	bytelen := end - start
	if bytelen != int(t.FixedSize()) {
		return nil, fmt.Errorf("%s: incorrect byte length", t.Name())
	}
	// assert that there are no 1s after the appropriate bitlen
	l0s := bits.LeadingZeros8(b[end-1])
	lastbytebitlen := int(t.Length) % 8
	if lastbytebitlen > 0 && lastbytebitlen+l0s < 8 {
		return nil, fmt.Errorf("%s: extraneous data", t.Name())
	}

	v := &BitArray{
		Data:   make([]byte, bytelen),
		Bitlen: t.Length,
	}
	copy(v.Data, b[start:end])
	return v, nil
}

func (t *BitVectorType) Deserialize(b []byte) (interface{}, error) {
	return t.DeserializeFrom(b, 0, len(b))
}
