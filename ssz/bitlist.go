package ssz

import (
	"fmt"
	"math/bits"
	"reflect"
)

type BitListType struct {
	Limit uint
}

func BitList(limit uint) SSZType {
	return &BitListType{
		Limit: limit,
	}
}

func (t *BitListType) Name() string {
	return fmt.Sprintf("BitList[%d]", t.Limit)
}

func (t *BitListType) Default() interface{} {
	return BitArray{
		Data:   make([]byte, 0),
		Bitlen: 0,
	}
}

func (t *BitListType) Type() reflect.Type {
	return reflect.TypeOf(t.Default())
}

func (t *BitListType) IsVariableSize() bool {
	return true
}

func (t *BitListType) FixedSize() uint {
	return 0
}

func (t *BitListType) Size(v interface{}) (uint, error) {
	var vv BitArray
	switch v := v.(type) {
	case *BitArray:
		vv = *v
	case BitArray:
		vv = v
	default:
		return 0, fmt.Errorf("%s: type mismatch: %v", t.Name(), reflect.TypeOf(v))
	}
	return (vv.Bitlen + 8) / 8, nil
}

func (t *BitListType) HashTreeRoot(v interface{}) ([32]byte, error) {
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
	return MixInLength(Merkleize(chunks, uint((t.Limit+255)/256)), vv.Bitlen), nil
}

func (t *BitListType) SerializeTo(v interface{}, b []byte, start int) error {
	var vv BitArray
	switch v := v.(type) {
	case *BitArray:
		vv = *v
	case BitArray:
		vv = v
	default:
		return fmt.Errorf("%s: type mismatch: %v", t.Name(), reflect.TypeOf(v))
	}
	if vv.Bitlen > t.Limit {
		return fmt.Errorf("%s: length exceeds limit: %d > %d", t.Name(), vv.Bitlen, t.Limit)
	}
	s := (int(vv.Bitlen) + 7) / 8
	copy(b[start:start+s], vv.Data[:])
	// set padding bit
	if vv.Bitlen%8 == 0 {
		b[start+s] = 1
	} else {
		b[start+s-1] |= 1 << (vv.Bitlen % 8)
	}
	return nil
}

func (t *BitListType) Serialize(v interface{}) ([]byte, error) {
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

func (t *BitListType) DeserializeFrom(b []byte, start, end int) (interface{}, error) {
	bytelen := end - start
	l0s := bits.LeadingZeros8(b[end-1])
	if l0s == 8 {
		return nil, fmt.Errorf("%s: last byte is all 0s", t.Name())
	}
	bitlen := uint(bytelen)*8 - uint(l0s) - 1
	if bitlen > t.Limit {
		return nil, fmt.Errorf("%s: length exceeds limit: %d > %d", t.Name(), bitlen, t.Limit)
	}
	v := &BitArray{
		Data:   make([]byte, (bitlen+7)/8),
		Bitlen: bitlen,
	}
	if bitlen%8 == 0 {
		copy(v.Data, b[start:end-1])
	} else {
		copy(v.Data, b[start:end])
		// remove padding bit
		v.Data[len(v.Data)-1] &= 0xff >> (l0s + 1)
	}
	return v, nil
}

func (t *BitListType) Deserialize(b []byte) (interface{}, error) {
	return t.DeserializeFrom(b, 0, len(b))
}
