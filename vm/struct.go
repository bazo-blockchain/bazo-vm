package vm

// Struct type represents the composite data type declaration that
// defines a group of variables.
type Struct Array

// NewStruct creates a new struct data structure.
func NewStruct(size int) Struct {
	array := NewArray()
	for i := 0; i < size; i++ {
		array.IncrementSize()
	}
	return Struct(array)
}
