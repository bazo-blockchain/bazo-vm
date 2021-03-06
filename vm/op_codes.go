package vm

// Supported Bazo OpCodes.
const (
	PushInt = iota
	PushBool
	PushChar
	PushStr
	Push
	Dup
	Roll
	Swap
	Pop
	Add
	Sub
	Mul
	Div
	Mod
	Exp
	Neg
	Eq
	NotEq
	Lt
	Gt
	LtEq
	GtEq
	ShiftL
	ShiftR
	BitwiseAnd
	BitwiseOr
	BitwiseXor
	BitwiseNot
	NoOp
	Jmp
	JmpTrue
	JmpFalse
	Call
	CallTrue
	CallExt
	Ret
	Size
	StoreLoc
	StoreSt
	LoadLoc
	LoadSt
	Address // Address of account
	Issuer  // Owner of smart contract account
	Balance // Balance of account
	Caller
	CallVal  // Amount of bazo coins transacted in transaction
	CallData // Parameters and function signature hash
	NewMap
	MapHasKey
	MapGetVal
	MapSetVal
	MapRemove
	NewArr
	ArrAppend
	ArrInsert
	ArrRemove
	ArrAt
	ArrLen
	NewStr
	StoreFld
	LoadFld
	SHA3
	CheckSig
	ErrHalt
	Halt
)

// Supported OpCode argument types
const (
	BYTES = iota + 1
	BYTE
	LABEL
	ADDR
)

// OpCode contains the code, name, number of arguments, argument types, gas price and gas factor of the opcode
type OpCode struct {
	code      byte
	Name      string
	Nargs     int
	ArgTypes  []int
	gasPrice  uint64
	gasFactor uint64
}

// OpCodes contains all OpCode definitions
var OpCodes = []OpCode{
	{PushInt, "pushint", 1, []int{BYTES}, 1, 1},
	{PushBool, "pushbool", 1, []int{BYTE}, 1, 1},
	{PushChar, "pushchar", 1, []int{BYTE}, 1, 1},
	{PushStr, "pushstr", 1, []int{BYTES}, 1, 1},
	{Push, "push", 1, []int{BYTES}, 1, 1},
	{Dup, "dup", 0, nil, 1, 2},
	{Roll, "roll", 1, []int{BYTE}, 1, 2},
	{Swap, "swap", 0, nil, 1, 2},
	{Pop, "pop", 0, nil, 1, 1},
	{Add, "add", 0, nil, 1, 2},
	{Sub, "sub", 0, nil, 1, 2},
	{Mul, "mult", 0, nil, 1, 2},
	{Div, "div", 0, nil, 1, 2},
	{Mod, "mod", 0, nil, 1, 2},
	{Exp, "exp", 0, nil, 1, 2},
	{Neg, "neg", 0, nil, 1, 2},
	{Eq, "eq", 0, nil, 1, 2},
	{NotEq, "neq", 0, nil, 1, 2},
	{Lt, "lt", 0, nil, 1, 2},
	{Gt, "gt", 0, nil, 1, 2},
	{LtEq, "lte", 0, nil, 1, 2},
	{GtEq, "gte", 0, nil, 1, 2},
	{ShiftL, "shiftl", 0, nil, 1, 2},
	{ShiftR, "shiftr", 0, nil, 1, 2},
	{BitwiseAnd, "bitwiseand", 0, nil, 1, 2},
	{BitwiseOr, "bitwiseor", 0, nil, 1, 2},
	{BitwiseXor, "bitwisexor", 0, nil, 1, 2},
	{BitwiseNot, "bitwisenot", 0, nil, 1, 2},
	{NoOp, "nop", 0, nil, 1, 1},
	{Jmp, "jmp", 1, []int{LABEL}, 1, 1},
	{JmpTrue, "jmptrue", 1, []int{LABEL}, 1, 1},
	{JmpFalse, "jmpfalse", 1, []int{LABEL}, 1, 1},
	{Call, "call", 2, []int{LABEL, BYTE}, 1, 1},
	{CallTrue, "callif", 2, []int{LABEL, BYTE}, 1, 1},
	{CallExt, "callext", 3, []int{ADDR, BYTE, BYTE, BYTE, BYTE, BYTE}, 1000, 2},
	{Ret, "ret", 0, nil, 1, 1},
	{Size, "size", 0, nil, 1, 1},
	{StoreLoc, "storeloc", 1, []int{BYTE}, 1, 2},
	{StoreSt, "storest", 1, []int{BYTE}, 1000, 2},
	{LoadLoc, "loadloc", 1, []int{BYTE}, 1, 2},
	{LoadSt, "loadst", 1, []int{BYTE}, 10, 2},
	{Address, "address", 0, nil, 1, 1},
	{Issuer, "issuer", 0, nil, 1, 1},
	{Balance, "balance", 0, nil, 1, 1},
	{Caller, "caller", 0, nil, 1, 1},
	{CallVal, "callval", 0, nil, 1, 1},
	{CallData, "calldata", 0, nil, 1, 1},
	{NewMap, "newmap", 0, nil, 1, 2},
	{MapHasKey, "maphaskey", 0, nil, 1, 2},
	{MapGetVal, "mapgetval", 0, nil, 1, 2},
	{MapSetVal, "mapsetval", 0, nil, 1, 2},
	{MapRemove, "mapremove", 0, nil, 1, 2},
	{NewArr, "newarr", 0, nil, 1, 2},
	{ArrAppend, "arrappend", 0, nil, 1, 2},
	{ArrInsert, "arrinsert", 0, nil, 1, 2},
	{ArrRemove, "arrremove", 0, nil, 1, 2},
	{ArrAt, "arrat", 0, nil, 1, 2},
	{ArrLen, "arrlen", 0, nil, 1, 2},
	{NewStr, "newstr", 1, []int{BYTE}, 1, 2},
	{StoreFld, "storefld", 1, []int{BYTE}, 1, 2},
	{LoadFld, "loadfld", 1, []int{BYTE}, 1, 2},
	{SHA3, "sha3", 0, nil, 1, 2},
	{CheckSig, "checksig", 0, nil, 1, 2},
	{ErrHalt, "errhalt", 0, nil, 0, 1},
	{Halt, "halt", 0, nil, 0, 1},
}
