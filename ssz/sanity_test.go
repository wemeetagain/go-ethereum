package ssz_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/ssz"
)

func TestSanity(t *testing.T) {
	t.Run("ByteVector", func(t *testing.T) {
		Bytes4 := ssz.ByteVector(4)

		b1 := Bytes4.Default().([]byte)
		b1[0] = 1
		s, err := Bytes4.Serialize(b1)
		if err != nil {
			t.Fatal(err)
		}
		b2, err := Bytes4.Deserialize(s)
		if err != nil {
			t.Fatal(err)
		}
		if !ssz.Equals(b1, b2) {
			t.Errorf("expected %v, got %v", b1, b2)
		}
		b3 := []byte{1, 0, 0, 0}
		ss, err := Bytes4.Serialize(b3)
		if err != nil {
			t.Fatal(err)
		}
		if !ssz.Equals(s, ss) {
			t.Errorf("expected %v, got %v", s, ss)
		}
	})
	t.Run("Vector", func(t *testing.T) {
		Bool4 := ssz.Vector(ssz.Bool(), 4)

		b1 := Bool4.Default().([]bool)
		b1[0] = true
		s, err := Bool4.Serialize(b1)
		if err != nil {
			t.Fatal(err)
		}
		b2, err := Bool4.Deserialize(s)
		if err != nil {
			t.Fatal(err)
		}
		if !ssz.Equals(b1, b2) {
			t.Errorf("expected %v, got %v", b1, b2)
		}
		b3 := []bool{true, false, false, false}
		ss, err := Bool4.Serialize(b3)
		if err != nil {
			t.Fatal(err)
		}
		if !ssz.Equals(s, ss) {
			t.Errorf("expected %v, got %v", s, ss)
		}
	})
	t.Run("Make", func(t *testing.T) {
		type Test struct {
			A bool
			B []byte `ssz:"List,4"`
		}
		TestType, err := ssz.MakeType(Test{}, nil)
		if err != nil {
			t.Fatal(err)
		}

		t.Log(TestType.Serialize(TestType.Default()))

		type Test2 struct {
			A bool
			B []byte `ssz:"Vector,4"`
		}
		TestType2, err := ssz.MakeType(Test2{}, nil)
		if err != nil {
			t.Fatal(err)
		}

		t.Log(TestType2.Serialize(TestType2.Default()))
	})
}
