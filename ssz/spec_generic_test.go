package ssz_test

import (
	"encoding/hex"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/ssz"
	"github.com/golang/snappy"
	"gopkg.in/yaml.v3"
)

func TestSszGeneric(t *testing.T) {
	testValidAndInvalidDir(
		t,
		"uints",
		"consensus-spec-tests/tests/general/phase0/ssz_generic/",
		func(parts []string) bool {
			// not planning on supporting uint128 and uint256
			return parts[1] == "128" || parts[1] == "256"
		},
		func(parts []string) (ssz.SSZType, error) {
			return strToType(fmt.Sprintf("uint%s", parts[1]))
		},
	)
	testValidAndInvalidDir(
		t,
		"boolean",
		"consensus-spec-tests/tests/general/phase0/ssz_generic/",
		func(parts []string) bool {
			return false
		},
		func(parts []string) (ssz.SSZType, error) {
			return strToType("bool")
		},
	)
	testValidAndInvalidDir(
		t,
		"basic_vector",
		"consensus-spec-tests/tests/general/phase0/ssz_generic/",
		func(parts []string) bool {
			// not planning on supporting uint128 and uint256, length 0 is illegal
			return parts[1] == "uint128" || parts[1] == "uint256" || parts[2] == "0"
		},
		func(parts []string) (ssz.SSZType, error) {
			l, err := strconv.Atoi(parts[2])
			if err != nil {
				t.Fatal(err)
			}
			var typ ssz.SSZType
			if parts[1] == "uint8" {
				typ = ssz.ByteVector(uint(l))
			} else {
				elemType, err := strToType(parts[1])
				if err != nil {
					t.Fatal(err)
				}
				typ = ssz.Vector(elemType, uint(l))
			}
			return typ, nil
		},
	)
	testValidAndInvalidDir(
		t,
		"bitvector",
		"consensus-spec-tests/tests/general/phase0/ssz_generic/",
		func(parts []string) bool {
			return parts[1] == "0"
		},
		func(parts []string) (ssz.SSZType, error) {
			l, err := strconv.Atoi(parts[1])
			if err != nil {
				t.Fatal(err)
			}
			return ssz.BitVector(uint(l)), nil
		},
	)
	testValidAndInvalidDir(
		t,
		"bitlist",
		"consensus-spec-tests/tests/general/phase0/ssz_generic/",
		func(parts []string) bool {
			return parts[1] == "no"
		},
		func(parts []string) (ssz.SSZType, error) {
			l, err := strconv.Atoi(parts[1])
			if err != nil {
				t.Fatal(err)
			}
			return ssz.BitList(uint(l)), nil
		},
	)
	testValidAndInvalidDir(
		t,
		"containers",
		"consensus-spec-tests/tests/general/phase0/ssz_generic/",
		func(s []string) bool {
			return false
		},
		func(parts []string) (ssz.SSZType, error) {
			return strToType(parts[0])
		},
	)
}

func testValidAndInvalidDir(t *testing.T, suiteName, d string, shouldSkip func([]string) bool, toType func([]string) (ssz.SSZType, error)) {
	testDir(t, suiteName+"/valid", d, shouldSkip, toType, testValid)
	testDir(t, suiteName+"/invalid", d, shouldSkip, toType, testInvalid)
}

func testDir(t *testing.T, suiteName, d string, shouldSkip func([]string) bool, toType func([]string) (ssz.SSZType, error), testFn func(*testing.T, string, ssz.SSZType)) {
	t.Run(suiteName, func(t *testing.T) {
		d = d + suiteName
		dir, err := os.ReadDir(d)
		if err != nil {
			t.Fatal(err)
		}

		for _, entry := range dir {
			name := entry.Name()
			parts := strings.Split(name, "_")
			if shouldSkip(parts) {
				continue
			}
			t.Run(entry.Name(), func(t *testing.T) {
				typ, err := toType(parts)
				if err != nil {
					t.Fatal(err)
				}
				testFn(t, d+"/"+entry.Name(), typ)
			})
		}
	})
}

func testValid(t *testing.T, dir string, typ ssz.SSZType) {
	eV, err := getValidValue(dir, typ)
	if err != nil {
		t.Fatal(err)
	}
	eHtr, err := getValidRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	eS, err := getSerialized(dir)
	if err != nil {
		t.Fatal(err)
	}

	// tests
	s, err := typ.Serialize(eV)
	if err != nil {
		t.Fatal(err)
	}

	htr, err := typ.HashTreeRoot(eV)
	if err != nil {
		t.Fatal(err)
	}

	v, err := typ.Deserialize(eS)
	if err != nil {
		t.Fatal(err)
	}

	if !bytesEqual(s, eS) {
		t.Errorf("%s serialized mismatch", typ.Name())
		t.Errorf("  actual: %x", s)
		t.Errorf("expected: %x", eS)
	}

	if htr != eHtr {
		t.Errorf("%s hash tree root mismatch", typ.Name())
		t.Errorf("  actual: %x", htr)
		t.Errorf("expected: %x", eHtr)
	}

	if !reflect.DeepEqual(v, eV) {
		t.Errorf("%s deserialized mismatch", typ.Name())
		t.Errorf("  actual: %v", v)
		t.Errorf("expected: %v", eV)
	}
}

func testInvalid(t *testing.T, dir string, typ ssz.SSZType) {
	eS, err := getSerialized(dir)
	if err != nil {
		t.Fatal(err)
	}

	v, err := typ.Deserialize(eS)
	if err == nil {
		t.Errorf("deserialize an invalid payload: %v", v)
	}
}

type SingleFieldTestStruct struct {
	A uint8
}

type SmallTestStruct struct {
	A uint16
	B uint16
}

type FixedTestStruct struct {
	A uint8
	B uint64
	C uint32
}

type VarTestStruct struct {
	A uint16
	B []uint16
	C uint8
}

type ComplexTestStruct struct {
	A uint16
	B []uint16
	C uint8
	D []uint8
	E VarTestStruct
	F []FixedTestStruct
	G []VarTestStruct
}

type BitsStruct struct {
	A ssz.BitArray
	B ssz.BitArray
	C ssz.BitArray
	D ssz.BitArray
	E ssz.BitArray
}

func strToTypeNoErr(s string) ssz.SSZType {
	t, _ := strToType(s)
	return t
}

func strToType(s string) (ssz.SSZType, error) {
	switch s {
	case "bool":
		return ssz.Bool(), nil
	case "uint8", "byte":
		return ssz.Uint(8), nil
	case "uint16":
		return ssz.Uint(16), nil
	case "uint32":
		return ssz.Uint(32), nil
	case "uint64":
		return ssz.Uint(64), nil
	case "SingleFieldTestStruct":
		return ssz.Container([]ssz.Field{{"A", ssz.Uint(8)}}, SingleFieldTestStruct{}), nil
	case "SmallTestStruct":
		return ssz.Container([]ssz.Field{{"A", ssz.Uint(16)}, {"B", ssz.Uint(16)}}, SmallTestStruct{}), nil
	case "FixedTestStruct":
		return ssz.Container([]ssz.Field{{"A", ssz.Uint(8)}, {"B", ssz.Uint(64)}, {"C", ssz.Uint(32)}}, FixedTestStruct{}), nil
	case "VarTestStruct":
		return ssz.Container([]ssz.Field{
			{"A", ssz.Uint(16)},
			{"B", ssz.List(ssz.Uint(16), 1024)},
			{"C", ssz.Uint(8)},
		}, VarTestStruct{}), nil
	case "ComplexTestStruct":
		return ssz.Container([]ssz.Field{
			{"A", ssz.Uint(16)},
			{"B", ssz.List(ssz.Uint(16), 128)},
			{"C", ssz.Uint(8)},
			{"D", ssz.ByteList(256)},
			{"E", strToTypeNoErr("VarTestStruct")},
			{"F", ssz.Vector(strToTypeNoErr("FixedTestStruct"), 4)},
			{"G", ssz.Vector(strToTypeNoErr("VarTestStruct"), 2)},
		}, ComplexTestStruct{}), nil
	case "BitsStruct":
		return ssz.Container([]ssz.Field{
			{"A", ssz.BitList(5)},
			{"B", ssz.BitVector(2)},
			{"C", ssz.BitVector(1)},
			{"D", ssz.BitList(6)},
			{"E", ssz.BitVector(8)},
		}, BitsStruct{}), nil
	}
	return nil, fmt.Errorf("unsupported type: %s", s)
}

type Meta struct {
	Root string `yaml:"root"`
}

// go-yaml is very loose about how it parses yaml into objects, possibly converting types :(
func toValue(typ ssz.SSZType, v interface{}) interface{} {
	if t, ok := typ.(*ssz.ContainerType); ok {
		vv := reflect.New(typ.Type()).Interface()
		fieldValues := make([]interface{}, len(t.FieldNames))
		v_ := v.(map[string]interface{})
		for i, name := range t.FieldNames {
			fieldValues[i] = toValue(t.FieldTypes[i], v_[name])
		}
		vv, _ = t.SetFieldValues(vv, fieldValues)
		return vv
	}
	if t, ok := typ.(*ssz.BitListType); ok {
		str := v.(string)
		b, _ := hex.DecodeString(str[2:])
		vv, _ := t.Deserialize(b)
		return vv
	}
	if t, ok := typ.(*ssz.BitVectorType); ok {
		str := v.(string)
		b, _ := hex.DecodeString(str[2:])
		vv, _ := t.Deserialize(b)
		return vv
	}
	if _, ok := typ.(*ssz.ByteVectorType); ok {
		vv := make([]byte, typ.FixedSize())
		v_ := v.([]interface{})
		for i, x := range v_ {
			vv[i] = byte(x.(int))
		}
		return vv
	}
	if _, ok := typ.(*ssz.ByteListType); ok {
		str := v.(string)
		b, _ := hex.DecodeString(str[2:])
		return b
	}
	if t, ok := typ.(*ssz.VectorType); ok {
		vv := typ.Default()
		v_ := v.([]interface{})
		for i, x := range v_ {
			reflect.ValueOf(vv).Index(i).Set(reflect.ValueOf(toValue(t.ElementType, x)))
		}
		return vv
	}
	if t, ok := typ.(*ssz.ListType); ok {
		vv := typ.Default()
		v_ := v.([]interface{})
		vvv := reflect.MakeSlice(reflect.TypeOf(vv), len(v_), len(v_))
		for i, x := range v_ {
			vvv.Index(i).Set(reflect.ValueOf(toValue(t.ElementType, x)))
		}
		return vvv.Interface()
	}
	if t, ok := typ.(*ssz.UintType); ok {
		switch t.Bitlen {
		case 8:
			x, ok := v.(int)
			if !ok {
				return uint8(v.(uint))
			}
			return uint8(x)
		case 16:
			x, ok := v.(int)
			if !ok {
				return uint16(v.(uint))
			}
			return uint16(x)
		case 32:
			x, ok := v.(int)
			if !ok {
				return uint32(v.(uint))
			}
			return uint32(x)
		case 64:
			x, ok := v.(int)
			if !ok {
				return uint64(v.(uint64))
			}
			return uint64(x)
		}
	}
	return v
}

func getValidValue(dir string, typ ssz.SSZType) (interface{}, error) {
	b, err := os.ReadFile(dir + "/value.yaml")
	if err != nil {
		return nil, err
	}
	// _, ok1 := typ.(*ssz.BitListType)
	// _, ok2 := typ.(*ssz.BitVectorType)
	// if ok1 || ok2 {
	// 	v := make([]bool, 0)
	// 	err = yaml.Unmarshal(b, &v)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	return ssz.MakeBitArrayFromBoolSlice(v), nil
	// }

	v := typ.Default()
	err = yaml.Unmarshal(b, &v)
	if err != nil {
		return nil, err
	}
	// post processing the yaml-parsed value to the correct type
	// somehow the yaml parser changes the type of v
	return toValue(typ, v), nil
}

func getValidRoot(dir string) ([32]byte, error) {
	b, err := os.ReadFile(dir + "/meta.yaml")
	if err != nil {
		return [32]byte{}, err
	}
	var m Meta
	err = yaml.Unmarshal(b, &m)
	if err != nil {
		return [32]byte{}, err
	}
	return hexStringToByteArray32(m.Root[2:])
}

func getSerialized(dir string) ([]byte, error) {
	b, err := os.ReadFile(dir + "/serialized.ssz_snappy")
	if err != nil {
		return nil, err
	}
	l, err := snappy.DecodedLen(b)
	if err != nil {
		return nil, err
	}
	s := make([]byte, l)
	s, err = snappy.Decode(s, b)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, x := range a {
		if x != b[i] {
			return false
		}
	}
	return true
}

func hexStringToByteArray32(hexStr string) ([32]byte, error) {
	// Decode the hex string
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to decode hex string: %w", err)
	}

	// Ensure the decoded length is exactly 32 bytes
	if len(decoded) != 32 {
		return [32]byte{}, fmt.Errorf("decoded byte array is not 32 bytes long")
	}

	// Copy the decoded bytes into a fixed-size array
	var byteArray [32]byte
	copy(byteArray[:], decoded)

	return byteArray, nil
}
