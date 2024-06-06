package ssz

import (
	"encoding/binary"
	"fmt"
	"reflect"
)

type UintType struct {
	Bitlen    uint
	fixedSize uint
}

func Uint(bitlen uint) SSZType {
	return &UintType{
		Bitlen:    bitlen,
		fixedSize: bitlen / 8,
	}
}

func (t *UintType) Name() string {
	switch t.Bitlen {
	case 8:
		return "uint8"
	case 16:
		return "uint16"
	case 32:
		return "uint32"
	case 64:
		return "uint64"
	default:
		return fmt.Sprintf("uint%d", t.Bitlen)
	}
}

func (t *UintType) Default() interface{} {
	switch t.Bitlen {
	case 8:
		return uint8(0)
	case 16:
		return uint16(0)
	case 32:
		return uint32(0)
	case 64:
		return uint64(0)
	default:
		return nil
	}
}

func (t *UintType) Max() uint64 {
	switch t.Bitlen {
	case 8:
		return uint64(^uint8(0))
	case 16:
		return uint64(^uint16(0))
	case 32:
		return uint64(^uint32(0))
	case 64:
		return ^uint64(0)
	default:
		return 0
	}

}

func (t *UintType) Type() reflect.Type {
	return reflect.TypeOf(t.Default())
}

func (t *UintType) IsVariableSize() bool {
	return false
}

func (t *UintType) FixedSize() uint {
	return t.fixedSize
}

func (t *UintType) Size(v interface{}) (uint, error) {
	return t.fixedSize, nil
}

func (t *UintType) HashTreeRoot(v interface{}) ([32]byte, error) {
	var b [32]byte
	err := t.SerializeTo(v, b[:], 0)
	if err != nil {
		return b, err
	}
	return b, nil
}

func (t *UintType) SerializeTo(v interface{}, b []byte, start int) error {
	val := reflect.ValueOf(v)
	var u uint
	switch val.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		if val.Uint() > t.Max() {
			return fmt.Errorf("%s: value out of range: %d", t.Name(), val.Int())
		}
		u = uint(val.Uint())
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		if val.Int() < 0 || uint64(val.Int()) > t.Max() {
			return fmt.Errorf("%s: value out of range: %d", t.Name(), val.Int())
		}
		u = uint(val.Int())
	default:
		return fmt.Errorf("%s: type mismatch: %s", t.Name(), val.Type())
	}

	switch t.Bitlen {
	case 8:
		b[start] = uint8(u)
	case 16:
		binary.LittleEndian.PutUint16(b[start:], uint16(u))
	case 32:
		binary.LittleEndian.PutUint32(b[start:], uint32(u))
	case 64:
		binary.LittleEndian.PutUint64(b[start:], uint64(u))
	default:
		return fmt.Errorf("%s: unsupported bitlen", t.Name())
	}
	return nil
}

func (t *UintType) Serialize(v interface{}) ([]byte, error) {
	b := make([]byte, t.FixedSize())
	err := t.SerializeTo(v, b, 0)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (t *UintType) DeserializeFrom(b []byte, start, end int) (interface{}, error) {
	if end-start != int(t.fixedSize) {
		return nil, fmt.Errorf("%s: length mismatch: %d != %d", t.Name(), end-start, t.fixedSize)
	}

	switch t.Bitlen {
	case 8:
		v := b[start]
		return v, nil
	case 16:
		v := binary.LittleEndian.Uint16(b[start:end])
		return v, nil
	case 32:
		v := binary.LittleEndian.Uint32(b[start:end])
		return v, nil
	case 64:
		v := binary.LittleEndian.Uint64(b[start:end])
		return v, nil
	default:
		return nil, fmt.Errorf("%s: unsupported bitlen", t.Name())
	}
}

func (t *UintType) Deserialize(b []byte) (interface{}, error) {
	return t.DeserializeFrom(b, 0, len(b))
}
