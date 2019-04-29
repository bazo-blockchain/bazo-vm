package vm

import (
	"github.com/pkg/errors"
)

// Struct type represents the composite data type declaration that
// defines a group of variables.
type Struct Array

// NewStruct creates a new struct data structure.
func newStruct(size uint16) Struct {
	array := NewArray()
	for i := uint16(0); i < size; i++ {
		_ = array.Append([]byte{0})
	}
	return Struct(array)
}

func structFromByteArray(arr []byte) (Struct, error) {
	array, err := ArrayFromByteArray(arr)
	if err != nil {
		return nil, err
	}

	return Struct(array), nil
}

func (s *Struct) toArray() *Array {
	return (*Array)(s)
}

// loadField returns the field at the given index
func (s *Struct) loadField(index uint16) ([]byte, error) {
	array := s.toArray()
	return array.At(index)
}

// storeField sets the element on the given index
func (s *Struct) storeField(index uint16, element []byte) error {
	array := s.toArray()
	size, err := array.getSize()
	if err != nil {
		return err
	}

	if index >= size {
		return errors.New("index out of bounds")
	}

	// Array insert does not work for an array with size = 1
	if size == index+1 {
		err := array.Remove(index)
		if err != nil {
			return err
		}
		err = array.Append(element)
		return err
	}
	return array.Insert(index, element)
}
