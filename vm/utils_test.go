package vm

import (
	"fmt"
	"gotest.tools/assert"
	"math/big"
	"testing"
)

func TestUtils_UInt16ToByteArray(t *testing.T) {
	ba := UInt16ToByteArray(0)

	if len(ba) != 2 {
		t.Errorf("Expected Byte Array with size 2 but got %v", len(ba))
	}

	var ui16max uint16 = 65535
	ba2 := UInt16ToByteArray(ui16max)

	if uint16(len(ba2)) != 2 {
		t.Errorf("Expected Byte Array with size 2 but got %v", uint16(len(ba2)))
	}
}

func TestUtils_ByteArrayToUI16(t *testing.T) {
	ba := []byte{0xFF, 0xFF}
	var ui16max uint16 = 65535

	r, err := ByteArrayToUI16(ba)

	if err != nil {
		t.Error(err)
	}

	if r != ui16max {
		t.Errorf("Expected result to be 65535 but was %v", r)
	}
}

func TestUtils_UI16AndByteArrayConversions(t *testing.T) {
	ba := UInt16ToByteArray(15)
	r, err := ByteArrayToUI16(ba)

	if err != nil {
		t.Error(err)
	}

	if r != 15 {
		t.Errorf("Expected result to be 15 but was %v", r)
	}

	ba2 := UInt16ToByteArray(65535)
	r2, err := ByteArrayToUI16(ba2)

	if err != nil {
		t.Error(err)
	}

	if r2 != 65535 {
		t.Errorf("Expected result to be 65535 but was %v", r)
	}
}

func TestUtils_IntToByteArrayAndBack(t *testing.T) {
	var start uint64 = 4651321
	ba := UInt64ToByteArray(start)

	end := ByteArrayToInt(ba)
	if start != uint64(end) {
		t.Errorf("Converstion from int to byteArray and back failed, start and end should be equal, are start: %v, end: %v", start, end)
	}
}

func TestUtils_StrToByteArrayAndBack(t *testing.T) {
	startStr := "asdf"
	ba := StrToBigInt(startStr)

	endStr := BigIntToString(ba)
	if startStr != endStr {
		t.Errorf("Converstion from str to byteArray and back failed, start and end should be equal, are start: %s, end: %s", startStr, endStr)
	}
}

func TestUtils_ByteArrayToInt(t *testing.T) {
	ba := []byte{0xA8, 0x93}

	expected := 43155
	actual := ByteArrayToInt(ba)
	if expected != actual {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}
}

func TestUtils_Uint16ToBigInt_Minimum(t *testing.T) {
	result := UInt16ToBigInt(0)
	assert.Equal(t, result.Cmp(big.NewInt(0)), 0)
}

func TestUtils_Uint16ToBigInt(t *testing.T) {
	var value uint16 = 1
	result := UInt16ToBigInt(value)
	assert.Equal(t, result.Cmp(big.NewInt(int64(1))), 0)
}

func TestUtils_Uint16ToBigInt_Maximum(t *testing.T) {
	var value = UINT16_MAX
	result := UInt16ToBigInt(value)
	assert.Equal(t, result.Cmp(big.NewInt(int64(UINT16_MAX))), 0)
}

func TestUtils_BigIntToUInt16_Negative(t *testing.T) {
	value := big.NewInt(-1)
	result, err := BigIntToUInt16(*value)
	assert.NilError(t, err)
	assert.Equal(t, result, uint16(1))
}

func TestUtils_BigIntToUInt16_Zero(t *testing.T) {
	value := big.NewInt(0)
	result, err := BigIntToUInt16(*value)
	assert.NilError(t, err)
	assert.Equal(t, result, uint16(0))
}

func TestUtils_BigIntToUInt16_Positive(t *testing.T) {
	value := big.NewInt(1)
	result, err := BigIntToUInt16(*value)
	assert.NilError(t, err)
	assert.Equal(t, result, uint16(1))
}

func TestUtils_BigIntToUInt16_Greater_Than_UInt16(t *testing.T) {
	value := big.NewInt(int64(UINT16_MAX) + 1)
	_, err := BigIntToUInt16(*value)
	assert.Equal(t, err.Error(), fmt.Sprintf("value cannot be greater than %v", UINT16_MAX))
}

// big.Int to uint
// ---------------

func TestUtils_BigIntToUInt_Zero(t *testing.T) {
	value := big.NewInt(0)
	result, err := BigIntToUInt(*value)
	assert.NilError(t, err)
	assert.Equal(t, result, uint(0))
}

func TestUtils_BigIntToUInt_Positive(t *testing.T) {
	value := big.NewInt(10)
	result, err := BigIntToUInt(*value)
	assert.NilError(t, err)
	assert.Equal(t, result, uint(10))
}

func TestUtils_BigIntToUInt_Negative(t *testing.T) {
	value := big.NewInt(-10)
	result, err := BigIntToUInt(*value)
	assert.NilError(t, err)
	assert.Equal(t, result, uint(10))
}

func TestUtils_BigIntToUInt_Max(t *testing.T) {
	var max uint = 4294967295
	value := big.NewInt(int64(max))
	result, err := BigIntToUInt(*value)
	assert.NilError(t, err)
	assert.Equal(t, result, max)
}

func TestUtils_BigIntToUInt_Overflow(t *testing.T) {
	max := int64(4294967295) + 1
	value := big.NewInt(max)
	_, err := BigIntToUInt(*value)

	assert.Equal(t, err.Error(), "value cannot be greater than 32bits")
}

// big.Int to []byte
// -----------------

func TestUtils_BigIntToByteArray_Zero(t *testing.T) {
	value := big.NewInt(0)
	result := BigIntToByteArray(*value)
	assertBytes(t, result, 0)
}

func TestUtils_BigIntToByteArray_Negative(t *testing.T) {
	value := big.NewInt(-2)
	result := BigIntToByteArray(*value)
	assertBytes(t, result, 1, 2)
}

func TestUtils_BigIntToByteArray_Positive(t *testing.T) {
	value := big.NewInt(1)
	result := BigIntToByteArray(*value)
	assertBytes(t, result, 0, 1)
}
