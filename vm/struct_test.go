package vm

import (
	"gotest.tools/assert"
	"testing"
)

func TestStruct_NewStruct(t *testing.T) {
	s := newStruct(2)
	size, err := s.fields.getSize()

	assert.NilError(t, err)
	assert.Equal(t, size, uint16(2))

	// Initial field value at index 0
	elementAt, atErr := s.fields.At(0)
	assert.NilError(t, atErr)
	assertBytes(t, elementAt, 0)

	// Initial field value at index 1
	elementAt, atErr = s.fields.At(1)
	assert.NilError(t, atErr)
	assertBytes(t, elementAt, 0)
}

func TestStruct_StoreField(t *testing.T) {
	s := newStruct(1)
	element := []byte{2}

	err := s.storeField(0, element)
	assert.NilError(t, err)

	fieldValue, loadErr := s.loadField(0)
	assert.NilError(t, loadErr)
	assertBytes(t, fieldValue, element...)
}

func TestStruct_StoreFields(t *testing.T) {
	s := newStruct(2)
	element1 := []byte{2}
	element2 := []byte{3}

	err := s.storeField(0, element1)
	assert.NilError(t, err)
	err = s.storeField(1, element2)
	assert.NilError(t, err)

	fieldValue, loadErr := s.loadField(0)
	assert.NilError(t, loadErr)
	assertBytes(t, fieldValue, element1...)

	fieldValue, loadErr = s.loadField(1)
	assert.NilError(t, loadErr)
	assertBytes(t, fieldValue, element2...)
}
