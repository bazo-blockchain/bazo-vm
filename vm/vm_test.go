package vm

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"testing"

	"fmt"

	"github.com/bazo-blockchain/bazo-miner/protocol"
	"gotest.tools/assert"
)

func TestVM_NewTestVM(t *testing.T) {
	vm := NewTestVM([]byte{})

	if len(vm.code) > 0 {
		t.Errorf("Actual code length is %v, should be 0 after initialization", len(vm.code))
	}

	if vm.pc != 0 {
		t.Errorf("Actual pc counter is %v, should be 0 after initialization", vm.pc)
	}
}

func TestVM_Exec_GasConsumption(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 8,
		PushInt, 1, 0, 8,
		Add,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.Fee = 30
	vm.context = mc

	success := vm.Exec(false)
	assert.Assert(t, success)

	ba, _ := vm.evaluationStack.Pop()
	expected := 16
	actual := ByteArrayToInt(ba)

	if expected != actual {
		t.Errorf("Expected first value to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_PushInt(t *testing.T) {
	code := []byte{
		PushInt, 0, // 0
		PushInt, 1, 1, 1, // -1
		PushInt, 1, 0, 255, // 255
		PushInt, 2, 0, 1, 0, // 256
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, isSuccess)

	expected := []int64{256, 255, -1, 0}

	for _, i := range expected {
		bint, _ := vm.PopSignedBigInt(OpCodes[PushInt])
		assert.Equal(t, bint.Cmp(big.NewInt(i)), 0)
	}
}

func TestVM_Exec_PushInt_OutOfBounds(t *testing.T) {
	code := []byte{
		PushInt, 1, 125,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}

	actual := string(tos)
	expected := "pushint: Instruction set out of bounds"

	if actual != expected {
		t.Errorf("Expected '%v' to be returned but got '%v'", expected, actual)
	}
}

func TestVM_Exec_PushBool(t *testing.T) {
	code := []byte{
		PushBool, 0,
		PushBool, 1,
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, isSuccess)

	tos, _ := vm.evaluationStack.Pop()
	assertBytes(t, tos, 1)
	tos, _ = vm.evaluationStack.Pop()
	assertBytes(t, tos, 0)
}

func TestVM_Exec_PushBool_Invalid(t *testing.T) {
	code := []byte{
		PushBool, 5,
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, !isSuccess)

	tos, _ := vm.evaluationStack.Pop()
	assert.Equal(t, string(tos), "pushbool: invalid bool value 5")
}

func TestVM_Exec_PushChar(t *testing.T) {
	code := []byte{
		PushChar, 104, // 'h'
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, isSuccess)

	tos, _ := vm.evaluationStack.Pop()
	assertBytes(t, tos, 104)
}

func TestVM_Exec_PushChar_Invalid(t *testing.T) {
	code := []byte{
		PushChar, 128,
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, !isSuccess)

	tos, _ := vm.evaluationStack.Pop()
	assert.Equal(t, string(tos), "pushchar: invalid ASCII code 128")
}

func TestVM_Exec_PushStr_Empty(t *testing.T) {
	code := []byte{
		PushStr, 0, // ""
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, isSuccess)

	tos, _ := vm.evaluationStack.Pop()
	assertBytes(t, tos)
}

func TestVM_Exec_PushStr(t *testing.T) {
	code := []byte{
		PushStr, 5, 104, 101, 108, 108, 111, // "hello"
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, isSuccess)

	tos, _ := vm.evaluationStack.Pop()
	assert.Equal(t, string(tos), "hello")
}

func TestVM_Exec_PushStr_Invalid(t *testing.T) {
	code := []byte{
		PushStr, 5, 104, 101, 200, 108, 111, // "hello"
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, !isSuccess)

	tos, _ := vm.evaluationStack.Pop()
	assert.Equal(t, string(tos), "pushstr: invalid ASCII code 200")
}

func TestVM_Exec_Push_Empty(t *testing.T) {
	code := []byte{
		Push, 0, // []
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, isSuccess)

	tos, _ := vm.evaluationStack.Pop()
	assertBytes(t, tos)
}

func TestVM_Exec_Push(t *testing.T) {
	code := []byte{
		Push, 2, 128, 255, // [128, 255]
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, isSuccess)

	tos, _ := vm.evaluationStack.Pop()
	assertBytes(t, tos, 128, 255)
}

func TestVM_Exec_Push_InvalidLength(t *testing.T) {
	code := []byte{
		Push, 2, 128, // [128]
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, !isSuccess)

	tos, _ := vm.evaluationStack.Pop()
	assert.Equal(t, string(tos), "push: Instruction set out of bounds")
}

func TestVM_Exec_Addition(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 125,
		PushInt, 2, 0, 168, 22,
		Add,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := 43155
	actual := ByteArrayToInt(tos)

	if expected != actual {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_Subtraction(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 6,
		PushInt, 1, 0, 3,
		Sub,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := 3
	actual := ByteArrayToInt(tos)

	if expected != actual {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_SubtractionWithNegativeResults(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 3,
		PushInt, 1, 0, 6,
		Sub,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := -3
	actual := ByteArrayToInt(tos[1:])

	if tos[0] == 0x01 {
		actual = actual * -1
	}

	if expected != actual {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_Multiplication(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 5,
		PushInt, 1, 0, 2,
		Mul,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := 10
	actual := ByteArrayToInt(tos)

	if expected != actual {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_Exponent(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 2,
		PushInt, 1, 0, 5,
		Exp,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := 25
	actual := ByteArrayToInt(tos)

	if expected != actual {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_Big_Exponent(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 8,
		PushInt, 1, 0, 5,
		Exp,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := 390625
	actual := ByteArrayToInt(tos)

	if expected != actual {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_Zero_Exponent(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 0,
		PushInt, 1, 0, 5,
		Exp,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := 1
	actual := ByteArrayToInt(tos)

	if expected != actual {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_Multiple_Exponent(t *testing.T) {
	// Calculates 5 ^ 3 ^ 2
	code := []byte{
		PushInt, 1, 0, 2,
		PushInt, 1, 0, 3,
		Exp,
		PushInt, 1, 0, 5,
		Exp,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := 1953125
	actual := ByteArrayToInt(tos)

	if expected != actual {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_Negative_Exponent(t *testing.T) {
	code := []byte{
		PushInt, 1, 1, 5,
		PushInt, 1, 0, 2,
		Exp,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := "exp: Negative exponents are not allowed."
	actual := string(tos)

	if expected != actual {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_Exponent_Out_of_Gas(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 100,
		PushInt, 1, 0, 1,
		Exp,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := "exp: Out of gas"
	actual := string(tos)

	if expected != actual {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_Modulo(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 5,
		PushInt, 1, 0, 2,
		Mod,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := 1
	actual := ByteArrayToInt(tos)

	if expected != actual {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_Negate(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 5,
		Neg,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := big.NewInt(-5)
	actual, _ := SignedBigIntConversion(tos, nil)

	if !(expected.Cmp(&actual) == 0) {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_Negate_True(t *testing.T) {
	code := []byte{
		PushBool, 1,
		Neg,
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, isSuccess)

	tos, _ := vm.evaluationStack.Pop()
	assertBytes(t, tos, 0)
}

func TestVM_Exec_Negate_False(t *testing.T) {
	code := []byte{
		PushBool, 0,
		Neg,
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, isSuccess)

	tos, _ := vm.evaluationStack.Pop()
	assertBytes(t, tos, 1)
}

func TestVM_Exec_Negate_Error(t *testing.T) {
	code := []byte{
		PushStr, 2, 3, 4,
		Neg,
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, !isSuccess)

	tos, _ := vm.evaluationStack.Pop()
	assert.Equal(t, string(tos), "neg: unable to negate 3")
}

func TestVM_Exec_Division(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 6,
		PushInt, 1, 0, 2,
		Div,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := 3
	actual := ByteArrayToInt(tos)

	if expected != actual {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_DivisionByZero(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 6,
		PushInt, 1, 0, 0,
		Div,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	result, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}

	expected := "div: Division by Zero"
	actual := string(result)
	if actual != expected {
		t.Errorf("Expected tos to be '%v' error message but was '%v'", expected, actual)
	}
}

func TestVM_Exec_Eq(t *testing.T) {
	code := []byte{
		Push, 3, 1, 0, 6,
		Push, 3, 1, 0, 6,
		Eq,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}

	if !ByteArrayToBool(tos) {
		t.Errorf("Actual value is %v, should be 1 after comparing 6 with 6", tos[0])
	}
}

func TestVM_Exec_Neq(t *testing.T) {
	code := []byte{
		Push, 3, 1, 0, 6,
		Push, 3, 1, 0, 5,
		NotEq,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}

	if !ByteArrayToBool(tos) {
		t.Errorf("Actual value is %v, should be 1 after comparing 6 with 5 to not be equal", tos[0])
	}
}

func TestVM_Exec_Lt(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 4,
		PushInt, 1, 0, 6,
		Lt,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, err := vm.evaluationStack.Pop()

	if err != nil {
		t.Errorf("%v", err)
	}

	if !ByteArrayToBool(tos) {
		t.Errorf("Actual value is %v, should be 1 after evaluating 4 < 6", tos[0])
	}
}

func TestVM_Exec_LtChar(t *testing.T) {
	code := []byte{
		PushChar, 0,
		PushChar, 70,
		Lt,
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, isSuccess)

	tos, _ := vm.evaluationStack.Pop()
	assertBytes(t, tos, 1)
}

func TestVM_Exec_LtChar_Negative(t *testing.T) {
	code := []byte{
		PushChar, 70,
		PushChar, 0,
		Lt,
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, isSuccess)

	tos, _ := vm.evaluationStack.Pop()
	assertBytes(t, tos, 0)
}

func TestVM_Exec_Gt(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 6,
		PushInt, 1, 0, 4,
		Gt,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}

	if !ByteArrayToBool(tos) {
		t.Errorf("Actual value is %v, should be 1 after evaluating 6 > 4", tos[0])
	}
}

func TestVM_Exec_GtChar(t *testing.T) {
	code := []byte{
		PushChar, 70,
		PushChar, 0,
		Gt,
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, isSuccess)

	tos, _ := vm.evaluationStack.Pop()
	assertBytes(t, tos, 1)
}

func TestVM_Exec_GtChar_Negative(t *testing.T) {
	code := []byte{
		PushChar, 0,
		PushChar, 70,
		Gt,
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, isSuccess)

	tos, _ := vm.evaluationStack.Pop()
	assertBytes(t, tos, 0)
}

func TestVM_Exec_Lte_islower(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 4,
		PushInt, 1, 0, 6,
		LtEq,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}

	if !ByteArrayToBool(tos) {
		t.Errorf("Actual value is %v, should be 1 after evaluating 4 <= 6", tos[0])
	}
}

func TestVM_Exec_Lte_isequals(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 6,
		PushInt, 1, 0, 6,
		LtEq,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}

	if !ByteArrayToBool(tos) {
		t.Errorf("Actual value is %v, should be 1 after evaluating 6 <= 6", tos[0])
	}
}

func TestVM_Exec_LtEq_Char(t *testing.T) {
	code := []byte{
		PushChar, 0,
		PushChar, 0,
		LtEq,
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, isSuccess)

	tos, _ := vm.evaluationStack.Pop()
	assertBytes(t, tos, 1)
}

func TestVM_Exec_Gte_isGreater(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 6,
		PushInt, 1, 0, 4,
		GtEq,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}

	if !ByteArrayToBool(tos) {
		t.Errorf("Actual value is %v, should be 1 after evaluating 6 >= 4", tos[0])
	}
}

func TestVM_Exec_Gte_isEqual(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 6,
		PushInt, 1, 0, 6,
		GtEq,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}

	if !ByteArrayToBool(tos) {
		t.Errorf("Actual value is %v, should be 1 after evaluating 6 >= 6", tos[0])
	}
}

func TestVM_Exec_GtEq_Char(t *testing.T) {
	code := []byte{
		PushChar, 70,
		PushChar, 70,
		GtEq,
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, isSuccess)

	tos, _ := vm.evaluationStack.Pop()
	assertBytes(t, tos, 1)
}

func TestVM_Exec_Shiftl(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 1,
		ShiftL, 3,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := 8
	actual := ByteArrayToInt(tos)

	if expected != actual {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_Shiftr(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 8,
		ShiftR, 3,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := 1
	actual := ByteArrayToInt(tos)

	if expected != actual {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_Jmptrue(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 3,
		PushInt, 1, 0, 4,
		Add,
		PushInt, 1, 0, 20,
		Lt,
		JmpTrue, 0, 21,
		Push, 1, 3,
		NoOp,
		NoOp,
		NoOp,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	if vm.evaluationStack.GetLength() != 0 {
		t.Errorf("After calling and returning, callStack lenght should be 0, but is %v", vm.evaluationStack.GetLength())
	}
}

func TestVM_Exec_Jmpfalse(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 3,
		PushInt, 1, 0, 4,
		Add,
		PushInt, 1, 0, 20,
		Gt,
		JmpFalse, 0, 21,
		Push, 1, 3,
		NoOp,
		NoOp,
		// JmpFalse jumps here
		NoOp,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	if vm.evaluationStack.GetLength() != 0 {
		t.Errorf("After calling and returning, evaluationStack lenght should be 0, but is %v", vm.evaluationStack.GetLength())
	}
}

func TestVM_Exec_Jmpfalse_Negative(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 3,
		PushInt, 1, 0, 4,
		Add,
		PushInt, 1, 0, 20,
		Lt,
		// Does not Jump
		JmpFalse, 0, 21,
		Push, 1, 3,
		NoOp,
		NoOp,
		NoOp,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	if vm.evaluationStack.GetLength() != 2 {
		t.Errorf("After calling and returning, evaluationStack lenght should be 2, but is %v", vm.evaluationStack.GetLength())
	}

	value, _ := vm.evaluationStack.PopIndexAt(0)
	result := uint(value[0])

	if result != 3 {
		t.Errorf("The value on the evaluationStack should be 3 but is %v", result)
	}
}

func TestVM_Exec_Jmp(t *testing.T) {
	code := []byte{
		Push, 1, 3,
		Jmp, 0, 14,
		Push, 1, 4,
		Add,
		Push, 1, 15,
		Add, // Jump here
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := 3
	actual := ByteArrayToInt(tos)

	if expected != actual {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_Call(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 10,
		PushInt, 1, 0, 8,
		Call, 0, 14, 2, 1,
		Halt,
		NoOp,
		NoOp,
		LoadLoc, 0, // Begin of called function at address 14
		LoadLoc, 1,
		Sub,
		Ret,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := 2
	actual := ByteArrayToInt(tos)

	if expected != actual {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}

	expected = 0
	actual = vm.callStack.GetLength()
	if expected != actual {
		t.Errorf("After calling and returning, callStack lenght should be %v, but was %v", expected, actual)
	}
}

func TestVM_Exec_Callif_true(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 10,
		PushInt, 1, 0, 8,
		PushInt, 1, 0, 10,
		PushInt, 1, 0, 10,
		Eq,
		CallTrue, 0, 25, 2, 1,
		Halt,
		NoOp,
		NoOp,
		LoadLoc, 0, // Begin of called function at address 20
		LoadLoc, 1,
		Sub,
		Ret,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := 2
	actual := ByteArrayToInt(tos)

	if expected != actual {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}

	expected = 0
	actual = vm.callStack.GetLength()
	if expected != actual {
		t.Errorf("After calling and returning, callStack lenght should be %v, but was %v", expected, actual)
	}
}

func TestVM_Exec_Callif_false(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 10,
		PushInt, 1, 0, 8,
		PushInt, 1, 0, 10,
		PushInt, 1, 0, 2,
		Eq,
		CallTrue, 0, 26, 2, 1,
		Halt,
		NoOp,
		NoOp,
		LoadLoc, 0, // Begin of called function at address 21
		LoadLoc, 1,
		Sub,
		Ret,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := 8
	actual := ByteArrayToInt(tos)

	if expected != actual {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}

	expected = 0
	actual = vm.callStack.GetLength()
	if expected != actual {
		t.Errorf("After skipping callif, callStack lenght should be '%v', but was '%v'", expected, actual)
	}
}

func TestVM_Exec_TosSize(t *testing.T) {
	code := []byte{
		PushInt, 2, 10, 4, 5,
		Size,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}

	expected := 3
	actual := ByteArrayToInt(tos)

	if expected != actual {
		t.Errorf("Expected element size to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_CallExt(t *testing.T) {
	code := []byte{
		Push, 1, 10,
		Push, 1, 8,
		CallExt, 227, 237, 86, 189, 8, 109, 137, 88, 72, 58, 18, 115, 79, 160, 174, 127, 92, 139, 177, 96, 239, 144, 146, 198, 126, 130, 237, 155, 25, 228, 199, 178, 41, 24, 45, 14, 2,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)
}

func TestVM_Exec_StoreLoc(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 1, // local variable x = 1
		PushInt, 1, 0, 2, // local variable y = 2
		Call, 0, 14, 2, // Call function with 2 variables (x & y)
		Halt,
		NoOp,
		PushInt, 1, 0, 4, // Function starts here at byte 14
		StoreLoc, 0, // Override local variable x with 4
		PushInt, 1, 0, 5,
		StoreLoc, 1, // override local variable y with 5
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, isSuccess)

	callstackTos, err := vm.callStack.Peek()
	assert.NilError(t, err)
	assert.Equal(t, len(callstackTos.variables), 2)

	assertBytes(t, callstackTos.variables[0], 0, 4)
	assertBytes(t, callstackTos.variables[1], 0, 5)
}

func TestVM_Exec_LoadSt(t *testing.T) {
	code := []byte{
		LoadSt, 1,
		LoadSt, 0,
		LoadSt, 2,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.ContractVariables = [][]byte{[]byte("Hi There!!"), {26}, {0}}
	vm.context = mc

	vm.Exec(false)

	expected := []byte{0}
	actual, _ := vm.evaluationStack.Pop()

	if !bytes.Equal(expected, actual) {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}

	result, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}

	expectedString := "Hi There!!"
	actualString := string(result)
	if expectedString != actualString {
		t.Errorf("The String on the Stack should be '%v' but was %v", expectedString, actualString)
	}

	expected = []byte{26}
	actual, _ = vm.evaluationStack.Pop()

	if !bytes.Equal(expected, actual) {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_StoreSt(t *testing.T) {
	code := []byte{
		PushInt, 9, 72, 105, 32, 84, 104, 101, 114, 101, 33, 33,
		StoreSt, 0,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.ContractVariables = [][]byte{[]byte("Something")}
	vm.context = mc
	mc.Fee = 100000
	vm.Exec(false)
	mc.PersistChanges()

	v, _ := vm.context.GetContractVariable(0)
	result := string(v)
	if result != "Hi There!!" {
		t.Errorf("The String on the Stack should be 'Hi There!!' but was '%v'", result)
	}
}

func TestVM_Exec_Address(t *testing.T) {
	code := []byte{
		Address,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	ba := [64]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	mc.Address = ba
	vm.context = mc

	vm.Exec(false)
	tos, _ := vm.evaluationStack.Pop()

	if len(tos) != 64 {
		t.Errorf("Expected TOS size to be 64, but got %v", len(tos))
	}

	//This just tests 1/8 of the address as Uint64 are 64 bits and the address is 64 bytes
	actual := binary.LittleEndian.Uint64(tos)
	var expected uint64 = 18446744073709551615

	if expected != actual {
		t.Errorf("Expected TOS size to be '%v', but got '%v'", expected, actual)
	}
}

func TestVM_Exec_Balance(t *testing.T) {
	code := []byte{
		Balance,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.Balance = uint64(100)
	vm.context = mc

	vm.Exec(false)
	tos, _ := vm.evaluationStack.Pop()

	if len(tos) != 8 {
		t.Errorf("Expected TOS size to be 64, but got %v", len(tos))
	}

	actual := binary.LittleEndian.Uint64(tos)
	var expected uint64 = 100

	if actual != expected {
		t.Errorf("Expected TOS to be '%v', but got '%v'", expected, actual)
	}
}

func TestVM_Exec_Caller(t *testing.T) {
	code := []byte{
		Caller,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	from := [32]byte{
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	}
	mc.From = from
	vm.context = mc

	vm.Exec(false)
	tos, _ := vm.evaluationStack.Pop()

	if len(tos) != 32 {
		t.Errorf("Expected TOS size to be 32, but got %v", len(tos))
	}

	if !bytes.Equal(tos, from[:]) {
		t.Errorf("Retrieved unexpected value")
	}
}

func TestVM_Exec_Callval(t *testing.T) {
	code := []byte{
		CallVal,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.Amount = uint64(100)
	vm.context = mc

	vm.Exec(false)
	tos, _ := vm.evaluationStack.Pop()

	if len(tos) != 8 {
		t.Errorf("Expected TOS size to be 8, but got %v", len(tos))
	}

	result := binary.LittleEndian.Uint64(tos)

	if result != 100 {
		t.Errorf("Expected value to be 100, but got %v", result)
	}
}

func TestVM_Exec_Calldata(t *testing.T) {
	code := []byte{
		CallData,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.Fee = 50

	td := []byte{
		1, 0x02,
		1, 0x05,
		4, 0x10, 0x12, 0x4, 0x12, // Function hash
	}
	mc.Data = td

	vm.context = mc
	vm.Exec(false)

	functionHash, _ := vm.evaluationStack.Pop()

	if !bytes.Equal(functionHash, td[5:]) {
		t.Errorf("expected '%# x' but got '%# x'", td[5:], functionHash)
	}

	arg1, _ := vm.evaluationStack.Pop()
	if !bytes.Equal(arg1, td[3:4]) {
		t.Errorf("expected '%# x' but got '%# x'", td[3:4], arg1)
	}

	arg2, _ := vm.evaluationStack.Pop()
	if !bytes.Equal(arg2, td[1:2]) {
		t.Errorf("expected '%# x' but got '%# x'", td[1:2], arg2)
	}
}

func TestVM_Exec_Sha3(t *testing.T) {
	code := []byte{
		Push, 1, 3,
		SHA3,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	actual, _ := vm.evaluationStack.Pop()
	expected := []byte{227, 237, 86, 189, 8, 109, 137, 88, 72, 58, 18, 115, 79, 160, 174, 127, 92, 139, 177, 96, 239, 144, 146, 198, 126, 130, 237, 155, 25, 228, 199, 178}
	if !bytes.Equal(actual, expected) {
		t.Errorf("Expected value to be \n '%v', \n but was \n '%v' \n after jumping to halt", expected, actual)
	}
}

func TestVM_Exec_Roll(t *testing.T) {
	code := []byte{
		Push, 1, 3,
		Push, 1, 4,
		Push, 1, 5,
		Push, 1, 6,
		Push, 1, 7,
		Roll, 2,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := 4
	actual := ByteArrayToInt(tos)
	if actual != expected {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_NewMap(t *testing.T) {
	code := []byte{
		NewMap,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	actual, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}

	expected := []byte{0x01, 0x00, 0x00}

	if !bytes.Equal(expected, actual) {
		t.Errorf("expected the Value of the new Map to be '[%v]' but was '[%v]'", expected, actual)
	}
}

func TestVM_Exec_MapHasKey_true(t *testing.T) {
	code := []byte{
		Push, 1, 1, //The key for MAPGETVAL

		Push, 2, 0x48, 0x48,
		Push, 1, 0x01,

		Push, 2, 0x69, 0x69,
		Push, 1, 0x02,

		Push, 2, 0x48, 0x69,
		Push, 1, 0x03,

		NewMap,

		MapPush,
		MapPush,
		MapPush,

		MapHasKey,

		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc

	exec := vm.Exec(false)
	if !exec {
		errorMessage, _ := vm.evaluationStack.Pop()
		t.Errorf("VM.Exec terminated with Error: %v", string(errorMessage))
	}

	tos, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}

	expected := true //Just for readability
	actual := ByteArrayToBool(tos)
	if expected != actual {
		t.Errorf("invalid value, Expected '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_MapHasKey_false(t *testing.T) {
	code := []byte{
		Push, 1, 0x06, //The key for MAPGETVAL

		Push, 2, 0x48, 0x48,
		Push, 1, 0x01,

		Push, 2, 0x69, 0x69,
		Push, 1, 0x02,

		Push, 2, 0x48, 0x69,
		Push, 1, 0x03,

		NewMap,

		MapPush,
		MapPush,
		MapPush,

		MapHasKey,

		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc

	exec := vm.Exec(false)
	if !exec {
		errorMessage, _ := vm.evaluationStack.Pop()
		t.Errorf("VM.Exec terminated with Error: %v", string(errorMessage))
	}

	tos, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}

	expected := false //Just for readability
	actual := ByteArrayToBool(tos)
	if expected != actual {
		t.Errorf("invalid value, Expected '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_MapPush(t *testing.T) {
	code := []byte{
		PushInt, 1, 72, 105,
		Push, 1, 0x03,
		NewMap,
		MapPush,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	exec := vm.Exec(false)

	if !exec {
		errorMessage, _ := vm.evaluationStack.Pop()
		t.Errorf("VM.Exec terminated with Error: %v", string(errorMessage))
	}

	m, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}

	mp, err2 := MapFromByteArray(m)
	if err2 != nil {
		t.Errorf("%v", err)
	}

	datastructure := mp[0]
	size, err := mp.getSize()

	if err != nil {
		t.Error(err)
	}

	if datastructure != 0x01 {
		t.Errorf("Invalid Datastructure ID, Expected 0x01 but was %v", datastructure)
	}

	if size != 1 {
		t.Errorf("invalid size, Expected 1 but was %v", size)
	}

}

func TestVM_Exec_MapGetVAL(t *testing.T) {
	code := []byte{
		Push, 1, 0x01, //The key for MAPGETVAL

		Push, 2, 0x48, 0x48,
		Push, 1, 0x01,

		Push, 2, 0x69, 0x69,
		Push, 1, 0x02,

		Push, 2, 0x48, 0x69,
		Push, 1, 0x03,

		NewMap,

		MapPush,
		MapPush,
		MapPush,

		MapGetVal,

		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.Fee = 300
	vm.context = mc

	exec := vm.Exec(false)
	if !exec {
		errorMessage, _ := vm.evaluationStack.Pop()
		t.Errorf("VM.Exec terminated with Error: %v", string(errorMessage))
	}

	actual, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}

	expected := []byte{72, 72}
	if !bytes.Equal(actual, expected) {
		t.Errorf("invalid value, Expected '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_MapSetVal(t *testing.T) {
	code := []byte{
		Push, 2, 0x55, 0x55, //Value to be reset by MAPSETVAL
		Push, 1, 0x03,

		Push, 2, 0x48, 0x69,
		Push, 1, 0x03,

		Push, 2, 0x69, 0x69,
		Push, 1, 0x02,

		NewMap,

		MapPush,
		MapPush,

		MapSetVal,

		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.Fee = 300
	vm.context = mc
	exec := vm.Exec(false)

	if !exec {
		errorMessage, _ := vm.evaluationStack.Pop()
		t.Errorf("VM.Exec terminated with Error: %v", string(errorMessage))
	}

	mbi, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}
	actual, err := MapFromByteArray(mbi)
	if err != nil {
		t.Errorf("%v", err)
	}

	expected := []byte{0x01,
		0x02, 0x00,
		0x01, 0x00, 0x02,
		0x02, 0x00, 0x69, 0x69,
		0x01, 0x00, 0x03,
		0x02, 0x00, 0x55, 0x55,
	}

	if !bytes.Equal(actual, expected) {
		t.Errorf("invalid datastructure, Expected '[%# x]' but was '[%# x]'", expected, actual)
	}
}

func TestVM_Exec_MapRemove(t *testing.T) {
	code := []byte{
		Push, 1, 0x03, // The Key to be removed with MAPREMOVE

		Push, 2, 0x48, 0x69,
		Push, 1, 0x03,

		Push, 2, 0x48, 0x48,
		Push, 1, 0x01,

		Push, 2, 0x69, 0x69,
		Push, 1, 0x02,

		NewMap,

		MapPush,
		MapPush,
		MapPush,

		MapRemove,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.Fee = 300
	vm.context = mc

	exec := vm.Exec(false)
	if !exec {
		errorMessage, _ := vm.evaluationStack.Pop()
		t.Errorf("VM.Exec terminated with Error: %v", string(errorMessage))
	}

	mapAsByteArray, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}

	actual, err := MapFromByteArray(mapAsByteArray)
	if err != nil {
		t.Errorf("%v", err)
	}

	expected := []byte{0x01,
		0x02, 0x00,
		0x01, 0x00, 0x02,
		0x02, 0x00, 0x69, 0x69,
		0x01, 0x00, 0x01,
		0x02, 0x00, 0x48, 0x48,
	}

	if !bytes.Equal(actual, expected) {
		t.Errorf("invalid datastructure, Expected '[%# x]' but was '[%# x]'", expected, actual)
	}
}

func TestVM_Exec_NewArr(t *testing.T) {
	code := []byte{
		NewArr,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	exec := vm.Exec(false)

	if !exec {
		errorMessage, _ := vm.evaluationStack.Pop()
		t.Errorf("VM.Exec terminated with Error: %v", string(errorMessage))
	}

	arr, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}
	expectedSize := []byte{0x00, 0x00}
	actualSize := arr[1:3]
	if !bytes.Equal(expectedSize, actualSize) {
		t.Errorf("invalid size, Expected %v but was '%v'", expectedSize, actualSize)
	}
}

func TestVM_Exec_ArrAppend(t *testing.T) {
	code := []byte{
		Push, 2, 0xFF, 0x00,
		NewArr,
		ArrAppend,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	exec := vm.Exec(false)
	if !exec {
		errorMessage, _ := vm.evaluationStack.Pop()
		t.Errorf("VM.Exec terminated with Error: %v", string(errorMessage))
	}

	arr, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}

	actual := arr[5:7]
	expected := []byte{0xFF, 0x00}
	if !bytes.Equal(expected, actual) {
		t.Errorf("invalid element appended, Expected '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_ArrInsert(t *testing.T) {
	code := []byte{
		Push, 2, 0x00, 0x02, // new value [0,2]
		Push, 2, 0x00, 0x00, // index 0

		Push, 1, 0xFE, // value [254] at index 1
		Push, 1, 0xFF, // value [255] at index 0
		NewArr,
		ArrAppend,
		ArrAppend,
		ArrInsert, // Replace [255] with the new value [0,2]
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.Fee = 300
	vm.context = mc
	exec := vm.Exec(false)
	if !exec {
		errorMessage, _ := vm.evaluationStack.Pop()
		t.Errorf("VM.Exec terminated with Error: %v", string(errorMessage))
	}

	actual, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}

	expectedSize := []byte{0x02}
	if !bytes.Equal(expectedSize, actual[1:2]) {
		t.Errorf("invalid element appended, Expected '[%# x]' but was '[%# x]'", expectedSize, actual[1:2])
	}

	expectedValue := []byte{0x00, 0x02}
	if !bytes.Equal(expectedValue, actual[5:7]) {
		t.Errorf("invalid element appended, Expected '[%# x' but was '[%# x]'", expectedValue, actual[5:7])
	}
}

func TestVM_Exec_ArrRemove(t *testing.T) {
	code := []byte{
		Push, 2, 0x01, 0x00, //Index of element to remove
		Push, 2, 0xBB, 0x00,
		Push, 2, 0xAA, 0x00,
		Push, 2, 0xFF, 0x00,

		NewArr,

		ArrAppend,
		ArrAppend,
		ArrAppend,
		ArrRemove,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.Fee = 300
	vm.context = mc
	exec := vm.Exec(false)

	if !exec {
		errorMessage, _ := vm.evaluationStack.Pop()
		t.Errorf("VM.Exec terminated with Error: %v", string(errorMessage))
	}

	a, err := vm.evaluationStack.Pop()
	if err != nil {
		t.Errorf("%v", err)
	}

	arr, bierr := ArrayFromByteArray(a)
	if bierr != nil {
		t.Errorf("%v", err)
	}

	size, err := arr.getSize()
	if err != nil {
		t.Error(err)
	}

	if size != uint16(2) {
		t.Errorf("invalid array size, Expected 2 but was '%v'", size)
	}

	expectedSecondElement := []byte{0xBB, 0x00}
	actualSecondElement, err2 := arr.At(uint16(1))
	if err2 != nil {
		t.Errorf("%v", err)
	}

	if !bytes.Equal(expectedSecondElement, actualSecondElement) {
		t.Errorf("invalid element on second index, Expected '[%# x]' but was '[%# x]'", expectedSecondElement, actualSecondElement)
	}
}

func TestVM_Exec_ArrAt(t *testing.T) {
	code := []byte{
		Push, 2, 0x02, 0x00, // index for ARRAT
		Push, 2, 0xBB, 0x00,
		Push, 2, 0xAA, 0x00,
		Push, 2, 0xFF, 0x00,

		NewArr,

		ArrAppend,
		ArrAppend,
		ArrAppend,

		ArrAt,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.Fee = 200
	vm.context = mc
	exec := vm.Exec(false)

	if !exec {
		errorMessage, _ := vm.evaluationStack.Pop()
		t.Errorf("VM.Exec terminated with Error: %v", string(errorMessage))
	}

	actual, err1 := vm.evaluationStack.Pop()

	if err1 != nil {
		t.Errorf("%v", err1)
	}

	expected := []byte{0xBB, 0x00}
	if !bytes.Equal(expected, actual) {
		t.Errorf("invalid element on first index, Expected '[%# x]' but was '[%# x]'", expected, actual)
	}

}

func TestVM_Exec_NewStr(t *testing.T) {
	code := []byte{
		NewStr, 2, 0, // size=2
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, isSuccess)

	arrBytes, err := vm.evaluationStack.Pop()
	assert.NilError(t, err)

	str, structErr := structFromByteArray(arrBytes)
	assert.NilError(t, structErr)
	assert.Assert(t, str != nil)

	arr := str.toArray()
	size, sizeErr := arr.getSize()
	assert.NilError(t, sizeErr)
	assert.Equal(t, size, uint16(2))
}

func TestVM_Exec_StoreFld(t *testing.T) {
	code := []byte{
		NewStr, 1, 0,
		PushInt, 1, 0, 4,
		StoreFld, 0, 0, // Store field on index 0
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, isSuccess)

	structBytes, err := vm.evaluationStack.Pop()
	assert.NilError(t, err)

	str, err := structFromByteArray(structBytes)
	assert.NilError(t, err)
	assert.Assert(t, str != nil)

	arr := str.toArray()
	element, err := arr.At(0)
	assert.NilError(t, err)
	assertBytes(t, element, 0, 4)
}

func TestVM_Exec_NonValidOpCode(t *testing.T) {
	code := []byte{
		89,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := "vm.exec(): Not a valid opCode"
	actual := string(tos)
	if actual != expected {
		t.Errorf("Expected error message to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_ArgumentsExceedInstructionSet(t *testing.T) {
	code := []byte{
		Push, 1, 0x00,
		Push, 0x0c, 0x01, 0x00, 0x03, 0x12, 0x05,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := "push: Instruction set out of bounds"
	actual := string(tos)
	if actual != expected {
		t.Errorf("Expected error message to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_PopOnEmptyStack(t *testing.T) {
	code := []byte{
		Push, 1, 0x01,
		SHA3,
		Sub, 0x02, 0x03,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	mc.Fee = 100
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := "sub: Invalid signing bit"
	actual := string(tos)
	if actual != expected {
		t.Errorf("Expected error message to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_FuzzReproduction_InstructionSetOutOfBounds(t *testing.T) {
	code := []byte{
		Push, 1, 20,
		Roll, 0,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := "vm.exec(): Instruction set out of bounds"
	actual := string(tos)
	if actual != expected {
		t.Errorf("Expected error message to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_FuzzReproduction_InstructionSetOutOfBounds2(t *testing.T) {
	code := []byte{
		CallExt, 231,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	mc.Fee = 100000
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := "callext: Instruction set out of bounds"
	actual := string(tos)
	if actual != expected {
		t.Errorf("Expected error message to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_FuzzReproduction_IndexOutOfBounds1(t *testing.T) {
	code := []byte{
		LoadSt, 0, 0, 33,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := "loadst: Index out of bounds"
	actual := string(tos)
	if actual != expected {
		t.Errorf("Expected error message to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_FuzzReproduction_IndexOutOfBounds2(t *testing.T) {
	code := []byte{
		PushInt, 4, 46, 110, 66, 50, 255, StoreSt, 123, 119,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	mc.Fee = 100000
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := "storest: Index out of bounds"
	actual := string(tos)
	if actual != expected {
		t.Errorf("Expected error message to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_FunctionCallSub(t *testing.T) {
	code := []byte{
		// start ABI
		CallData,
		Dup,
		PushInt, 1, 0, 1,
		Eq,
		JmpTrue, 0, 20,
		Dup,
		PushInt, 1, 0, 2,
		Eq,
		JmpTrue, 0, 23,
		Halt,
		// end ABI
		Pop,
		Sub,
		Halt,
		Pop,
		Add,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)

	mc.Data = []byte{
		2, 0, 5,
		2, 0, 2,
		2, 0, 1, // Function hash
	}

	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := 3
	actual := ByteArrayToInt(tos)
	if actual != expected {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_FunctionCall(t *testing.T) {
	code := []byte{
		// start ABI
		CallData,
		Dup,
		PushInt, 1, 0, 1,
		Eq,
		JmpTrue, 0, 20,
		Dup,
		PushInt, 1, 0, 2,
		Eq,
		JmpTrue, 0, 23,
		Halt,
		// end ABI
		Pop,
		Sub,
		Halt,
		Pop,
		Add,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)

	mc.Data = []byte{
		2, 0, 2,
		2, 0, 5,
		2, 0, 2, // Function hash
	}

	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := 7
	actual := ByteArrayToInt(tos)
	if actual != expected {
		t.Errorf("Expected result to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_GithubIssue13(t *testing.T) {
	code := []byte{
		Address, ArrAt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := "arrat: pop() on empty stack"
	actual := string(tos)
	if actual != expected {
		t.Errorf("Expected error message to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_FuzzReproduction_ContextOpCode1(t *testing.T) {
	code := []byte{
		Caller, Caller, ArrAppend,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.Fee = 200
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := "arrappend: not a valid array"
	actual := string(tos)
	if actual != expected {
		t.Errorf("Expected error message to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_FuzzReproduction_ContextOpCode2(t *testing.T) {
	code := []byte{
		Address, Caller, ArrAppend,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.Fee = 200
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := "arrappend: not a valid array"
	actual := string(tos)
	if actual != expected {
		t.Errorf("Expected error message to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_Exec_FuzzReproduction_EdgecaseLastOpcodePlusOne(t *testing.T) {
	code := []byte{
		Halt + 1,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := "vm.exec(): Not a valid opCode"
	actual := string(tos)
	if actual != expected {
		t.Errorf("Expected error message to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_PopBytes(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 8,
		PushInt, 1, 0, 8,
		Add,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.Fee = 11
	vm.context = mc

	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := 16
	actual := ByteArrayToInt(tos)
	if actual != expected {
		t.Errorf("Expected ToS to be '%v' but was '%v'", expected, actual)
	}

	expectedFee := 4
	actualFee := vm.fee

	if int(actualFee) != expectedFee {
		t.Errorf("Expected actual fee to be '%v' but was '%v'", expected, actual)
	}
}

func TestVM_FuzzTest_Reproduction(t *testing.T) {
	code := []byte{
		42, 0, 11, 1, 155, 6, 4, 13, 80, 89, 144, 14, 178, 188, 176, 41, 215, 171, 74, 28, 97, 232, 200, 151, 211, 147, 185, 143, 13, 220, 87, 77, 33, 223, 218, 249, 39, 126, 162, 59, 136, 178, 192, 120, 189, 37, 32, 37, 99, 130, 12, 145, 66, 131, 252, 30, 213, 1, 193, 101, 2, 15, 216, 19, 252, 78, 121, 20, 24, 216,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.Fee = 11
	vm.context = mc

	vm.Exec(false)
}

func TestVM_FuzzTest_Reproduction_IndexOutOfRange(t *testing.T) {
	code := []byte{
		36, 16, 19, 33, 46, 55, 188,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.Fee = 100
	vm.context = mc

	vm.Exec(false)
}

func TestVM_GasCalculation(t *testing.T) {
	code := []byte{
		PushInt, 64, 0, 8, 179, 91, 9, 9, 6, 136, 231, 56, 7, 146, 99, 170, 98, 183, 40, 118, 185, 95,
		106, 14, 143, 25, 99, 79, 76, 222, 197, 5, 218, 90, 216, 47, 218, 74, 53, 139, 62, 28, 104,
		180, 139, 65, 103, 193, 244, 169, 85, 39, 160, 218, 158, 207, 118, 37, 78, 42, 186, 64, 4, 70, 70, 190, 177,
		PushInt, 1, 0, 8,
		Add,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.Fee = 11
	vm.context = mc

	vm.Exec(false)

	expectedFee := 2
	actualFee := vm.fee

	if int(actualFee) != expectedFee {
		t.Errorf("Expected actual fee to be '%v' but was '%v'", expectedFee, actualFee)
	}
}

func TestVM_PopBytesOutOfGas(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 8,
		PushInt, 1, 0, 8,
		Add,
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.Fee = 3
	vm.context = mc

	vm.Exec(false)

	tos, _ := vm.evaluationStack.Pop()

	expected := "add: Out of gas"
	actual := string(tos)
	if actual != expected {
		t.Errorf("Expected ToS to be '%v' but was '%v'", expected, actual)
	}

	expectedFee := 0
	actualFee := vm.fee

	if int(actualFee) != expectedFee {
		t.Errorf("Expected actual fee to be '%v' but was '%v'", expected, actual)
	}
}

func BenchmarkVM_Exec_ModularExponentiation_GoImplementation(b *testing.B) {
	benchmarks := []struct {
		name string
		bLen int
	}{
		{"bIs32B", 32},
		{"bIs128B", 128},
		{"bIs255B", 255},
	}

	var base big.Int
	var exponent big.Int
	var modulus big.Int

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {

				base.SetBytes(protocol.RandomBytesWithLength(bm.bLen))
				exponent.SetBytes(protocol.RandomBytesWithLength(1))
				modulus.SetBytes(protocol.RandomBytesWithLength(2))

				modularExpGo(base, exponent, modulus)
			}

			b.ReportAllocs()
		})
	}
}

func BenchmarkVM_Exec_ModularExponentiation_ContractImplementation(b *testing.B) {
	benchmarks := []struct {
		name string
		bLen int
	}{
		{"bIs32B", 32},
		{"bIs128B", 128},
		{"bIs255B", 255},
	}

	var base big.Int
	var exponent big.Int
	var modulus big.Int

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				base.SetBytes(protocol.RandomBytesWithLength(bm.bLen))
				exponent.SetBytes(protocol.RandomBytesWithLength(1))
				modulus.SetBytes(protocol.RandomBytesWithLength(2))

				contract := modularExpContract(base, exponent, modulus)

				vm := NewTestVM([]byte{})
				mc := NewMockContext(contract)
				mc.Fee = 1000000000000
				vm.context = mc

				if vm.Exec(false) != true {
					tos, err := vm.evaluationStack.Pop()
					fmt.Println(string(tos), err)
					b.Fail()
				}
				vm.pc = 0
				mc.Fee = 10000000000000
			}

			b.ReportAllocs()
			fmt.Println(b.Name())
		})
	}
}

func modularExpGo(base big.Int, exponent big.Int, modulus big.Int) *big.Int {
	if modulus.Cmp(big.NewInt(0)) == 0 {
		return big.NewInt(0)
	}
	start := big.NewInt(1)
	c := big.NewInt(1)
	for i := new(big.Int).Set(start); i.Cmp(&exponent) < 0; i.Add(i, big.NewInt(1)) {
		c = c.Mul(c, &base)
		c = c.Mod(c, &modulus)
	}
	return c
}

func modularExpContract(base big.Int, exponent big.Int, modulus big.Int) []byte {
	baseVal := BigIntToPushableBytes(base)
	exponentVal := BigIntToPushableBytes(exponent)
	modulusVal := BigIntToPushableBytes(modulus)

	addressBeforeExp := UInt16ToByteArray(uint16(39) + uint16(len(baseVal)) + uint16(len(modulusVal)))
	addressAfterExp := UInt16ToByteArray(uint16(66) + uint16(len(baseVal)) + uint16(len(modulusVal)) + uint16(len(exponentVal)))
	addressForLoop := UInt16ToByteArray(uint16(20) + uint16(len(baseVal)) + uint16(len(modulusVal)) + uint16(len(exponentVal)))

	contract := []byte{
		PushInt,
	}
	contract = append(contract, baseVal...)
	contract = append(contract, PushInt)
	contract = append(contract, modulusVal...)
	contract = append(contract, []byte{
		Dup,
		PushInt, 1, 0, 0,
		Eq,
		JmpTrue,
	}...)
	contract = append(contract, addressBeforeExp[1])
	contract = append(contract, addressBeforeExp[0])
	contract = append(contract, []byte{
		PushInt, 1, 0, 1, // Counter (c)
		PushInt, 1, 0, 0, //i
		PushInt,
	}...)
	contract = append(contract, exponentVal...)
	contract = append(contract, []byte{
		//LOOP start
		//Duplicate arguments
		Roll, 2,
		Dup, //Stack: [[0 11 75] [0 11 75] [0 13] [0 0] [0 1] [0 4]]
		Roll, 4,
		Dup, // STACK Stack: [[04] [0 4] [0 11 75] [0 11 75] [0 13] [0 0] [0 1]]
		// PUT in order
		Roll, 1, //Stack: [[0 11 75] [0 4] [0 4] [0 11 75] [0 13] [0 0] [0 1]]
		Roll, 4, //Stack: [[0 0] [0 11 75] [0 4] [0 4] [0 11 75] [0 13] [0 1]]
		Roll, 4, //Stack: [[0 13] [0 0] [0 11 75] [0 4] [0 4] [0 11 75] [0 1]]
		Roll, 3, //Stack: [[0 4] [0 13] [0 0] [0 11 75] [0 4] [0 11 75] [0 1]]
		Roll, 4, //Stack: [[0 11 75] [0 4] [0 13] [0 0] [0 11 75] [0 4] [0 1]]
		Roll, 5, //Stack: [[0 1] [0 11 75] [0 4] [0 13] [0 0] [0 11 75] [0 4]]
		// Order: counter, modulus, base, exp, i, modulus, base
		Call,
	}...)
	contract = append(contract, byte(addressAfterExp[1]))
	contract = append(contract, byte(addressAfterExp[0]))
	contract = append(contract, []byte{
		3,
		// PUT in order
		Roll, 1,
		Roll, 1,

		// Order: exp, i - counter, modulus, base,
		Dup,
		Roll, 1,
		PushInt, 1, 0, 1,
		Add,
		Dup,
		Roll, 1,
		Roll, 1,
		Roll, 2,
		Lt,
		JmpTrue,
	}...)
	contract = append(contract, addressForLoop[1])
	contract = append(contract, addressForLoop[0])
	contract = append(contract, []byte{
		// LOOP END
		Halt,

		// FUNCTION Order: c, modulus, base,
		LoadLoc, 2,
		LoadLoc, 0,
		Mul,
		LoadLoc, 1,
		Mod,
		Ret,
	}...)

	return contract
}

func TestVm_Exec_Loop(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 0, //i
		PushInt, 1, 0, 13, // Exp

		// Order: exp, i
		Dup,
		Roll, 1,
		PushInt, 1, 0, 1,
		Add,
		Dup,
		Roll, 1,
		Roll, 1,
		Roll, 2,
		Lt,
		JmpTrue, 0, 8, // Adjust address
		// LOOP END
		Halt,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.Fee = 1000
	vm.context = mc
	vm.Exec(false)

	expected := 13
	actual, _ := vm.evaluationStack.Pop()

	if ByteArrayToInt(actual[1:]) != expected {
		t.Errorf("Expected actual result to be '%v' but was '%v'", expected, actual)
	}
}

func TestVm_Exec_ModularExponentiation_ContractImplementation(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 4, // Base 4
		PushInt, 2, 0, 1, 241, // Modulus 497

		// Address 9
		// IF modulus equals 0
		Dup,
		PushInt, 0,
		Eq,
		JmpTrue, 0, 42, // Adjust address

		// Address 16
		PushInt, 1, 0, 1, // Counter (c)
		PushInt, 0, // i
		PushInt, 1, 0, 13, // Exp

		// Address 26
		//LOOP start: Stack: [[0 13] [0] [0 1] [0 1 241] [0 4]]
		Roll, 2,
		//Duplicate arguments
		Dup, //Stack: [[0 11 75] [0 11 75] [0 13] [0] [0 1] [0 4]]
		Roll, 4,
		Dup, // STACK Stack: [[04] [0 4] [0 11 75] [0 11 75] [0 13] [0 0] [0 1]]
		// PUT in order
		Roll, 1, //Stack: [[0 11 75] [0 4] [0 4] [0 11 75] [0 13] [0 0] [0 1]]
		Roll, 4, //Stack: [[0 0] [0 11 75] [0 4] [0 4] [0 11 75] [0 13] [0 1]]
		Roll, 4, //Stack: [[0 13] [0 0] [0 11 75] [0 4] [0 4] [0 11 75] [0 1]]
		Roll, 3, //Stack: [[0 4] [0 13] [0 0] [0 11 75] [0 4] [0 11 75] [0 1]]
		Roll, 4, //Stack: [[0 11 75] [0 4] [0 13] [0 0] [0 11 75] [0 4] [0 1]]
		Roll, 5, //Stack: [[0 1] [0 11 75] [0 4] [0 13] [0 0] [0 11 75] [0 4]]

		// Address 44
		// Order: counter, modulus, base, exp, i, modulus, base
		Call, 0, 73, 3, 5,
		// PUT in order
		Roll, 1,
		Roll, 1,

		// Address 53
		// Order: exp, i - counter, modulus, base,
		Dup,
		Roll, 1,
		PushInt, 1, 0, 1,
		Add,
		Dup,
		Roll, 1,
		Roll, 1,
		Roll, 2,
		Lt,
		JmpTrue, 0, 26, // Adjust address
		// LOOP END
		Halt,

		// Address 73
		// FUNCTION Order: c, modulus, base,
		LoadLoc, 2,
		LoadLoc, 0,
		Mul,
		LoadLoc, 1,
		Mod,
		Ret,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.Fee = 1000
	vm.context = mc
	vm.Exec(true)

	expected := 445
	vm.evaluationStack.Pop()
	vm.evaluationStack.Pop()
	actual, _ := vm.evaluationStack.Pop()

	if ByteArrayToInt(actual[1:]) != expected {
		t.Errorf("Expected actual result to be '%v' but was '%v'", expected, actual)
	}
}

func TestMultipleReturnValues(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 1,
		PushInt, 1, 0, 2,
		Call, 0, 14, 2, 2,
		Halt,
		NoOp,
		NoOp,
		LoadLoc, 0, // Begin of called function at address 14
		LoadLoc, 1,
		Ret,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.Fee = 1000
	vm.context = mc
	vm.Exec(false)

	firstExpected := 2
	secondExpected := 1
	firstActual, _ := vm.evaluationStack.Pop()
	secondActual, _ := vm.evaluationStack.Pop()

	if firstActual == nil || secondActual == nil {
		t.Error("Function did not return enough values.")
	}

	if ByteArrayToInt(firstActual[1:]) != firstExpected || ByteArrayToInt(secondActual[1:]) != secondExpected {
		t.Errorf("Actual return values '%v' and '%v' do not match with expected values '%v' and '%v'",
			ByteArrayToInt(firstActual[1:]),
			ByteArrayToInt(secondActual[1:]),
			firstExpected,
			secondExpected,
		)
	}
}

func TestMultipleReturnValuesDifferentTypes(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 1,
		PushBool, 0,
		Call, 0, 14, 2, 2,
		Halt,
		NoOp,
		NoOp,
		LoadLoc, 0, // Begin of called function at address 14
		LoadLoc, 1,
		Ret,
	}

	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	mc.Fee = 1000
	vm.context = mc
	vm.Exec(false)

	firstExpected := false
	secondExpected := 1
	firstActual, _ := vm.evaluationStack.Pop()
	secondActual, _ := vm.evaluationStack.Pop()

	if firstActual == nil || secondActual == nil {
		t.Error("Function did not return enough values.")
	}

	if ByteArrayToBool(firstActual) != firstExpected || ByteArrayToInt(secondActual[1:]) != secondExpected {
		t.Errorf("Actual return values '%v' and '%v' do not match with expected values '%v' and '%v'",
			ByteArrayToInt(firstActual[1:]),
			ByteArrayToInt(secondActual[1:]),
			firstExpected,
			secondExpected,
		)
	}
}

func TestPeekEvalStack(t *testing.T) {
	code := []byte{
		PushInt, 1, 0, 2, // [128]
		PushBool, 0,
		Push, 4, 1, 2, 3, 4,
		Halt,
	}

	vm, isSuccess := execCode(code)
	assert.Assert(t, isSuccess)

	evalStack := vm.PeekEvalStack()
	assert.Equal(t, len(evalStack), 3)
	assertBytes(t, evalStack[0], 0, 2)
	assertBytes(t, evalStack[1], 0)
	assertBytes(t, evalStack[2], 1, 2, 3, 4)
}

// Helper functions
// ----------------

func execCode(code []byte) (*VM, bool) {
	vm := NewTestVM([]byte{})
	mc := NewMockContext(code)
	vm.context = mc
	isSuccess := vm.Exec(false)

	return &vm, isSuccess
}

func assertBytes(t *testing.T, actual []byte, expected ...byte) {
	assert.Equal(t, len(actual), len(expected))

	for i, b := range actual {
		assert.Equal(t, b, expected[i])
	}
}
