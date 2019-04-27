package vm

import (
	"github.com/pkg/errors"
)

// Struct type represents the composite data type declaration that
// defines a group of variables.
type Struct struct {
	fields Array
	size   uint16
}

// NewStruct creates a new struct data structure.
func newStruct(size uint16) *Struct {
	array := NewArray()
	for i := uint16(0); i < size; i++ {
		_ = array.Append([]byte{0})
	}
	return &Struct{array, size}
}

// LoadField returns the field at the given index
func (s *Struct) LoadField(index uint16) ([]byte, error) {
	return s.fields.At(index)
}

// StoreField sets the element on the given index
func (s *Struct) StoreField(index uint16, element []byte) error {
	if index >= s.size {
		return errors.New("index out of bounds")
	}

	// Array insert does not work for an array with size = 1
	if s.size == index+1 {
		err := s.fields.Remove(index)
		if err != nil {
			return err
		}
		err = s.fields.Append(element)
		return err
	}
	return s.fields.Insert(index, element)
}
