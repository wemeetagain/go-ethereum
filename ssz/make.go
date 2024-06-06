package ssz

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

const (
	ID_BitList = iota
	ID_BitVector
	ID_List
	ID_Vector
)

var annotationTypeMap = map[string]int{
	"BitList":   ID_BitList,
	"BitVector": ID_BitVector,
	"List":      ID_List,
	"Vector":    ID_Vector,
}

type Annotation struct {
	Type int
	L    uint
}

func MakeType(v interface{}, annotation *Annotation) (SSZType, error) {
	vv := reflect.ValueOf(v)
	if vv.Kind() == reflect.Ptr {
		vv = vv.Elem()
	}

	switch vv.Kind() {
	case reflect.Bool:
		return Bool(), nil
	case reflect.Uint8:
		return Uint(8), nil
	case reflect.Uint16:
		return Uint(16), nil
	case reflect.Uint32:
		return Uint(32), nil
	case reflect.Uint64:
		return Uint(64), nil
	case reflect.Slice:
		if annotation == nil {
			return nil, fmt.Errorf("missing annotation for slice")
		}
		if annotation.Type == ID_List {
			if reflect.TypeOf(v).Elem().Kind() == reflect.Uint8 {
				return ByteList(annotation.L), nil
			} else {
				elemType, err := MakeType(vv.Index(0).Interface(), nil)
				if err != nil {
					return nil, err
				}
				return List(elemType, annotation.L), nil
			}
		}
		if annotation.Type == ID_Vector {
			if reflect.TypeOf(v).Elem().Kind() == reflect.Uint8 {
				return ByteVector(annotation.L), nil
			} else {
				elemType, err := MakeType(vv.Index(0).Interface(), nil)
				if err != nil {
					return nil, err
				}
				return Vector(elemType, annotation.L), nil
			}
		}
		return nil, fmt.Errorf("unsupported annotation type for slice: %v", annotation.Type)

	case reflect.Struct:
		if vv.Type().Name() == "BitArray" {
			if annotation == nil {
				return nil, fmt.Errorf("missing annotation for BitArray")
			}
			if annotation.Type == ID_BitList {
				return BitList(annotation.L), nil
			}
			if annotation.Type == ID_BitVector {
				return BitVector(annotation.L), nil
			}
			return nil, fmt.Errorf("unsupported annotation type for BitArray")
		}
		t := reflect.TypeOf(v)
		fields := make([]Field, 0, vv.NumField())
		for i := 0; i < vv.NumField(); i++ {
			tag := t.Field(i).Tag.Get("ssz")
			fieldType, err := MakeType(vv.Field(i).Interface(), parseTag(tag))
			if err != nil {
				return nil, err
			}
			fields = append(fields, Field{t.Field(i).Name, fieldType})
		}
		return Container(fields, v), nil

	}
	return nil, fmt.Errorf("unsupported type: %v", vv.Kind())
}

func parseTag(tag string) *Annotation {
	if tag == "" {
		return nil
	}
	var t Annotation
	parts := strings.Split(tag, ",")
	t.Type = annotationTypeMap[parts[0]]
	if len(parts) > 1 {
		l, _ := strconv.Atoi(parts[1])
		t.L = uint(l)
	}
	return &t
}
