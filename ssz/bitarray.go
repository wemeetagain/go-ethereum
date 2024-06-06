package ssz

import "fmt"

type BitArray struct {
	Data   []byte
	Bitlen uint
}

func MakeBitArrayFromBitlen(bitlen uint) *BitArray {
	return &BitArray{
		Data:   make([]byte, (bitlen+7)/8),
		Bitlen: bitlen,
	}
}

func MakeBitArrayFromBoolSlice(v []bool) *BitArray {
	ba := MakeBitArrayFromBitlen(uint(len(v)))
	for i, b := range v {
		if b {
			ba.Data[i/8] |= 1 << (i % 8)
		}
	}
	return ba
}
func (ba *BitArray) GetBit(i uint) bool {
	mask := byte(1 << (i % 8))
	return ba.Data[i/8]&mask == mask
}

func (ba *BitArray) SetBit(i uint, b bool) error {
	if i >= ba.Bitlen {
		return fmt.Errorf("index out of range: %d", i)
	}
	mask := byte(1 << (i % 8))
	if b {
		ba.Data[i/8] |= mask
	} else {
		ba.Data[i/8] &= ^mask
	}
	return nil
}

func (ba *BitArray) GetBoolSlice() []bool {
	v := make([]bool, ba.Bitlen)
	for i := uint(0); i < ba.Bitlen; i++ {
		v[i] = ba.GetBit(i)
	}
	return v
}
