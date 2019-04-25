package vm

import (
	"gotest.tools/assert"
	"testing"
)

func TestStruct_NewStruct(t *testing.T) {
	s := NewStruct(2)
	a := Array(s)
	size, err := a.getSize()

	assert.NilError(t, err)
	assert.Equal(t, size, uint16(2))
}
