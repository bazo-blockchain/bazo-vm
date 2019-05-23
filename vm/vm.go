package vm

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"golang.org/x/crypto/sha3"
)

// Context is the VM execution context
// which is composed with data coming from the transaction and the account.
// Context interface declares functions required to start the execution of the contract.
type Context interface {
	GetContract() []byte
	GetContractVariable(index int) ([]byte, error)
	SetContractVariable(index int, value []byte) error
	GetAddress() [64]byte
	GetIssuer() [32]byte
	GetBalance() uint64
	GetSender() [32]byte
	GetAmount() uint64
	GetTransactionData() []byte
	GetFee() uint64
	GetSig1() [64]byte
}

// VM is a stack-based virtual machine and executes the contract code sequentially.
type VM struct {
	code            []byte
	pc              int // Program counter
	fee             uint64
	evaluationStack *Stack
	callStack       *CallStack
	context         Context
}

// NewVM creates a new Bazo virtual machine with the context received from Bazo miner.
func NewVM(context Context) VM {
	return VM{
		code:            []byte{},
		pc:              0,
		fee:             0,
		evaluationStack: NewStack(),
		callStack:       NewCallStack(),
		context:         context,
	}
}

// NewTestVM creates a new Bazo virtual machine with the test contract code.
func NewTestVM(byteCode []byte) VM {
	return VM{
		code:            []byte{},
		pc:              0,
		fee:             0,
		evaluationStack: NewStack(),
		callStack:       NewCallStack(),
		context:         NewMockContext(byteCode),
	}
}

// Private function, that can be activated by Exec call, useful for debugging
func (vm *VM) trace() {
	stack := vm.evaluationStack
	addr := vm.pc

	byteCode := int(vm.code[vm.pc])
	if len(OpCodes) <= byteCode {
		stack.Push([]byte("Trace: invalid opcode "))
		return
	}
	opCode := OpCodes[byteCode]

	var args []byte
	var formattedArgs string
	var counter int

	for _, argType := range opCode.ArgTypes {
		switch argType {
		case BYTES:
			if len(vm.code)-vm.pc > 0 {
				nargs := int(vm.code[vm.pc+1])

				if nargs < (len(vm.code)-vm.pc)-1 {
					args = vm.code[vm.pc+2+counter : vm.pc+nargs+3+counter]
					counter += nargs
					formattedArgs = formattedArgs + fmt.Sprintf("%v (bytes) ", args[:])
				}
			}

		case BYTE:
			if len(vm.code)-vm.pc > 0 {
				args = []byte{vm.code[vm.pc+1+counter]}
				counter++
				formattedArgs += fmt.Sprintf("%v (byte) ", args[:])
			}

		case ADDR:
			if len(vm.code)-vm.pc > 31 {
				args = vm.code[vm.pc+1+counter : vm.pc+33+counter]
				counter += 32
				formattedArgs += fmt.Sprintf("%v (bazo address) ", args[:])
			}

		case LABEL:
			if len(vm.code)-vm.pc > 1 {
				args = vm.code[vm.pc+1+counter : vm.pc+3+counter]
				counter += 2
				formattedArgs += fmt.Sprintf("%v (address) ", ByteArrayToInt(args[:]))
			}
		}
	}

	reversedStack := make([][]byte, stack.GetLength())
	maxIndex := len(stack.Stack) - 1
	for i := maxIndex; i >= 0; i-- {
		reversedStack[maxIndex-i] = stack.Stack[i]
	}

	fmt.Printf("\t  Stack: %v \n", reversedStack)
	fmt.Printf("\t  %v of max. %v Bytes in use \n", stack.memoryUsage, stack.memoryMax)
	fmt.Printf("⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅⋅\n")
	fmt.Printf("%04d: %-6s %v \n", addr, opCode.Name, formattedArgs)
}

// Exec executes the contract code and stores the result on evaluation stack.
func (vm *VM) Exec(trace bool) bool {
	vm.code = vm.context.GetContract()
	vm.fee = vm.context.GetFee()

	if len(vm.code) > 100000 {
		vm.evaluationStack.Push([]byte("vm.exec(): Instruction set to big"))
		return false
	}

	// Infinite Loop until return called
	for {
		if trace {
			vm.trace()
		}

		// Fetch
		byteCode, err := vm.fetch("vm.exec()")
		if err != nil {
			vm.evaluationStack.Push([]byte("vm.exec(): " + err.Error()))
			return false
		}

		// Return false if instruction is not an opCode
		if len(OpCodes) <= int(byteCode) {
			vm.evaluationStack.Push([]byte("vm.exec(): Not a valid opCode"))
			return false
		}

		opCode := OpCodes[byteCode]
		// Subtract gas used for operation
		if vm.fee < opCode.gasPrice {
			vm.evaluationStack.Push([]byte("vm.exec(): out of gas"))
			return false
		}
		vm.fee -= opCode.gasPrice

		// Decode
		switch opCode.code {

		case PushInt:
			totalBytes, errArg1 := vm.fetch(opCode.Name)
			if !vm.checkErrors(opCode.Name, errArg1) {
				return false
			}

			var err error
			if totalBytes == 0 {
				err = vm.evaluationStack.Push([]byte{0})
			} else {
				// Amount of bytes pushed (including sign byte)
				// Maximum amount of bytes that can be pushed is 256
				byteCount := int(totalBytes) + 1 //
				bytes, errArg2 := vm.fetchMany(opCode.Name, byteCount)

				if !vm.checkErrors(opCode.Name, errArg2) {
					return false
				}

				err = vm.evaluationStack.Push(bytes)
			}

			if err != nil {
				_ = vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}
		case PushBool:
			boolValue, err := vm.fetch(opCode.Name)

			if !vm.checkErrors(opCode.Name, err) {
				return false
			}

			if boolValue > 1 {
				_ = vm.evaluationStack.Push([]byte(
					fmt.Sprintf("%s: invalid bool value %v", opCode.Name, boolValue)))
				return false
			}

			err = vm.evaluationStack.Push([]byte{boolValue})
			if !vm.checkErrors(opCode.Name, err) {
				return false
			}
		case PushChar:
			charCode, err := vm.fetch(opCode.Name)

			if !vm.checkErrors(opCode.Name, err) {
				return false
			}

			if charCode > 127 {
				_ = vm.evaluationStack.Push([]byte(
					fmt.Sprintf("%s: invalid ASCII code %v", opCode.Name, charCode)))
				return false
			}

			err = vm.evaluationStack.Push([]byte{charCode})
			if !vm.checkErrors(opCode.Name, err) {
				return false
			}
		case PushStr:
			length, errArg1 := vm.fetch(opCode.Name)
			bytes, errArg2 := vm.fetchMany(opCode.Name, int(length))

			if !vm.checkErrors(opCode.Name, errArg1, errArg2) {
				return false
			}

			for _, charCode := range bytes {
				if charCode > 127 {
					_ = vm.evaluationStack.Push([]byte(
						fmt.Sprintf("%s: invalid ASCII code %v", opCode.Name, charCode)))
					return false
				}
			}

			err = vm.evaluationStack.Push(bytes)
			if !vm.checkErrors(opCode.Name, err) {
				return false
			}
		case Push:
			length, errArg1 := vm.fetch(opCode.Name)
			bytes, errArg2 := vm.fetchMany(opCode.Name, int(length))

			if !vm.checkErrors(opCode.Name, errArg1, errArg2) {
				return false
			}

			err = vm.evaluationStack.Push(bytes)
			if !vm.checkErrors(opCode.Name, err) {
				return false
			}
		case Dup:
			tos, err := vm.PopBytes(opCode)

			if !vm.checkErrors(opCode.Name, err) {
				return false
			}

			err = vm.evaluationStack.Push(tos)

			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			err = vm.evaluationStack.Push(tos)

			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case Roll:
			arg, err := vm.fetch(opCode.Name) // arg shows how many have to be rolled
			index := vm.evaluationStack.GetLength() - (int(arg) + 2)

			if !vm.checkErrors(opCode.Name, err) {
				return false
			}

			if index != -1 {
				if int(arg) >= vm.evaluationStack.GetLength() {
					vm.evaluationStack.Push([]byte(opCode.Name + ": index out of bounds"))
					return false
				}

				newTos, err := vm.evaluationStack.PopIndexAt(index)

				if err != nil {
					vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
					return false
				}

				err = vm.evaluationStack.Push(newTos)

				if err != nil {
					vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
					return false
				}
			}
		case Swap:
			last, err1 := vm.evaluationStack.Pop()
			secondLast, err2 := vm.evaluationStack.Pop()
			if !vm.checkErrors(opCode.Name, err1, err2) {
				return false
			}

			err1 = vm.evaluationStack.Push(last)
			err2 = vm.evaluationStack.Push(secondLast)
			if !vm.checkErrors(opCode.Name, err1, err2) {
				return false
			}
		case Pop:
			_, rerr := vm.PopBytes(opCode)
			if !vm.checkErrors(opCode.Name, rerr) {
				return false
			}

		case Add:
			isSuccess := vm.evaluateBigIntOperation(opCode, func(left *big.Int, right *big.Int) {
				left.Add(left, right)
			})

			if !isSuccess {
				return false
			}
		case Sub:
			isSuccess := vm.evaluateBigIntOperation(opCode, func(left *big.Int, right *big.Int) {
				left.Sub(left, right)
			})

			if !isSuccess {
				return false
			}
		case Mul:
			isSuccess := vm.evaluateBigIntOperation(opCode, func(left *big.Int, right *big.Int) {
				left.Mul(left, right)
			})

			if !isSuccess {
				return false
			}

		case Exp:

			left, rerr := vm.PopSignedBigInt(opCode)
			right, lerr := vm.PopSignedBigInt(opCode)

			if !vm.checkErrors(opCode.Name, rerr, lerr) {
				return false
			}

			if right.Cmp(big.NewInt(0)) == -1 {
				vm.evaluationStack.Push([]byte(opCode.Name + ": Negative exponents are not allowed."))
				return false
			}

			// The Exp OpCode is a special case in terms of gas calculation. The calculation of the gasCost is done
			// during execution. An Exp function such as 2 ** n can be split up into n multiplications of the first
			// factor -> 2 * 2 * 2 ... (n times). Therefore the gasCosts need to be as high as if the user performed
			// n multiplications. As the user already paid the opcode price, we reduce the gasCost by this price.
			gasCost := opCode.gasPrice*uint64(right.Int64()) - opCode.gasPrice

			if int64(vm.fee-gasCost) < 0 {
				vm.evaluationStack.Push([]byte(opCode.Name + ": Out of gas"))
				return false
			}

			left.Exp(&left, &right, nil)

			err := vm.evaluationStack.Push(SignedByteArrayConversion(left))

			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case Div:
			right, rerr := vm.PopSignedBigInt(opCode)
			left, lerr := vm.PopSignedBigInt(opCode)

			if !vm.checkErrors(opCode.Name, rerr, lerr) {
				return false
			}

			if right.Cmp(big.NewInt(0)) == 0 {
				vm.evaluationStack.Push([]byte(opCode.Name + ": Division by Zero"))
				return false
			}

			left.Div(&left, &right)
			err := vm.evaluationStack.Push(SignedByteArrayConversion(left))

			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case Mod:
			right, rerr := vm.PopSignedBigInt(opCode)
			left, lerr := vm.PopSignedBigInt(opCode)

			if !vm.checkErrors(opCode.Name, rerr, lerr) {
				return false
			}

			if right.Cmp(big.NewInt(0)) == 0 {
				vm.evaluationStack.Push([]byte(opCode.Name + ": Division by Zero"))
				return false
			}

			left.Mod(&left, &right)
			err := vm.evaluationStack.Push(SignedByteArrayConversion(left))

			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case Neg:
			tos, err := vm.PopBytes(opCode)

			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			switch tos[0] {
			case 1:
				tos[0] = 0
			case 0:
				tos[0] = 1
			default:
				err = fmt.Errorf("unable to negate %v", tos[0])
				_ = vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			err = vm.evaluationStack.Push(tos)
			if err != nil {
				_ = vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}
		case Eq:
			right, rerr := vm.PopBytes(opCode)
			left, lerr := vm.PopBytes(opCode)

			if !vm.checkErrors(opCode.Name, rerr, lerr) {
				return false
			}

			result := bytes.Compare(left, right)
			err := vm.evaluationStack.Push(BoolToByteArray(result == 0))

			if err != nil {
				_ = vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}
		case NotEq:
			right, rerr := vm.PopBytes(opCode)
			left, lerr := vm.PopBytes(opCode)

			if !vm.checkErrors(opCode.Name, rerr, lerr) {
				return false
			}

			result := bytes.Compare(left, right)
			err := vm.evaluationStack.Push(BoolToByteArray(result != 0))

			if err != nil {
				_ = vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}
		case Lt:
			isSuccess := vm.evaluateRelationalComp(opCode, -1)
			if !isSuccess {
				return false
			}
		case Gt:
			isSuccess := vm.evaluateRelationalComp(opCode, 1)
			if !isSuccess {
				return false
			}
		case LtEq:
			isSuccess := vm.evaluateRelationalComp(opCode, -1, 0)
			if !isSuccess {
				return false
			}
		case GtEq:
			isSuccess := vm.evaluateRelationalComp(opCode, 0, 1)
			if !isSuccess {
				return false
			}
		case ShiftL:
			shiftsBigInt, err := vm.PopSignedBigInt(opCode)
			tos, errStack := vm.PopSignedBigInt(opCode)

			if !vm.checkErrors(opCode.Name, err, errStack) {
				return false
			}

			nrOfShifts, err := BigIntToUInt(shiftsBigInt)
			if !vm.checkErrors(opCode.Name, err) {
				return false
			}

			tos.Lsh(&tos, nrOfShifts)
			err = vm.evaluationStack.Push(SignedByteArrayConversion(tos))

			if err != nil {
				_ = vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case ShiftR:
			shiftsBigInt, err := vm.PopSignedBigInt(opCode)
			tos, errStack := vm.PopSignedBigInt(opCode)

			if !vm.checkErrors(opCode.Name, err, errStack) {
				return false
			}

			nrOfShifts, err := BigIntToUInt(shiftsBigInt)
			if !vm.checkErrors(opCode.Name, err) {
				return false
			}

			tos.Rsh(&tos, nrOfShifts)
			err = vm.evaluationStack.Push(SignedByteArrayConversion(tos))

			if err != nil {
				_ = vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}
		case BitwiseAnd:
			isSuccess := vm.evaluateBigIntOperation(opCode, func(left *big.Int, right *big.Int) {
				left.And(left, right)
			})

			if !isSuccess {
				return false
			}
		case BitwiseOr:
			isSuccess := vm.evaluateBigIntOperation(opCode, func(left *big.Int, right *big.Int) {
				left.Or(left, right)
			})

			if !isSuccess {
				return false
			}
		case BitwiseXor:
			isSuccess := vm.evaluateBigIntOperation(opCode, func(left *big.Int, right *big.Int) {
				left.Xor(left, right)
			})

			if !isSuccess {
				return false
			}
		case BitwiseNot:
			bigInt, err := vm.PopSignedBigInt(opCode)
			if !vm.checkErrors(opCode.Name, err) {
				return false
			}

			bigInt.Not(&bigInt)
			err = vm.evaluationStack.Push(SignedByteArrayConversion(bigInt))

			if !vm.checkErrors(opCode.Name, err) {
				return false
			}

		case NoOp:
			_, err := vm.fetch(opCode.Name)

			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case Jmp:
			nextInstruction, err := vm.fetchMany(opCode.Name, 2)

			if !vm.checkErrors(opCode.Name, err) {
				return false
			}

			var jumpTo big.Int
			jumpTo.SetBytes(nextInstruction)

			vm.pc = int(jumpTo.Int64())

		case JmpTrue:
			nextInstruction, errArg := vm.fetchMany(opCode.Name, 2)
			right, errStack := vm.PopBytes(opCode)
			if !vm.checkErrors(opCode.Name, errArg, errStack) {
				return false
			}

			if ByteArrayToBool(right) {
				vm.pc = ByteArrayToInt(nextInstruction)
			}

		case JmpFalse:
			nextInstruction, errArg := vm.fetchMany(opCode.Name, 2)
			right, errStack := vm.PopBytes(opCode)
			if !vm.checkErrors(opCode.Name, errArg, errStack) {
				return false
			}

			if !ByteArrayToBool(right) {
				vm.pc = ByteArrayToInt(nextInstruction)
			}

		case Call:
			returnAddressBytes, errArg1 := vm.fetchMany(opCode.Name, 2) // Shows where to jump after executing
			argsToLoad, errArg2 := vm.fetch(opCode.Name)                // Shows how many elements have to be popped from evaluationStack
			nrOfReturnTypesByte, errArg3 := vm.fetch(opCode.Name)

			if !vm.checkErrors(opCode.Name, errArg1, errArg2, errArg3) {
				return false
			}

			var returnAddress big.Int
			returnAddress.SetBytes(returnAddressBytes)

			if int(returnAddress.Int64()) == 0 || int(returnAddress.Int64()) > len(vm.code) {
				_ = vm.evaluationStack.Push([]byte(opCode.Name + ": ReturnAddress out of bounds"))
				return false
			}

			nrOfReturnTypes := int(nrOfReturnTypesByte)

			if nrOfReturnTypes < 0 {
				_ = vm.evaluationStack.Push([]byte(opCode.Name + ": Number of return types cannot be negative"))
				return false
			}

			frame := &Frame{
				returnAddress:   vm.pc,
				variables:       make(map[int][]byte),
				nrOfReturnTypes: nrOfReturnTypes,
			}

			for i := int(argsToLoad) - 1; i >= 0; i-- {
				frame.variables[i], err = vm.PopBytes(opCode)
				if err != nil {
					_ = vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
					return false
				}
			}
			frame.evalStackOffset = len(vm.evaluationStack.Stack)

			vm.callStack.Push(frame)
			vm.pc = int(returnAddress.Int64())

		case CallTrue:
			returnAddressBytes, errArg1 := vm.fetchMany(opCode.Name, 2) // Shows where to jump after executing
			argsToLoad, errArg2 := vm.fetch(opCode.Name)                // Shows how many elements have to be popped from evaluationStack
			nrOfReturnTypesByte, errArg3 := vm.fetch(opCode.Name)
			right, errStack := vm.PopBytes(opCode)

			if !vm.checkErrors(opCode.Name, errArg1, errArg2, errArg3, errStack) {
				return false
			}

			if ByteArrayToBool(right) {
				var returnAddress big.Int
				returnAddress.SetBytes(returnAddressBytes)

				if int(returnAddress.Int64()) == 0 || int(returnAddress.Int64()) > len(vm.code) {
					_ = vm.evaluationStack.Push([]byte(opCode.Name + ": ReturnAddress out of bounds"))
					return false
				}

				nrOfReturnTypes := int(nrOfReturnTypesByte)

				if nrOfReturnTypes < 0 {
					_ = vm.evaluationStack.Push([]byte(opCode.Name + ": Number of return types cannot be negative"))
					return false
				}

				frame := &Frame{
					returnAddress:   vm.pc,
					variables:       make(map[int][]byte),
					nrOfReturnTypes: nrOfReturnTypes,
				}

				for i := int(argsToLoad) - 1; i >= 0; i-- {
					frame.variables[i], err = vm.PopBytes(opCode)
					if err != nil {
						_ = vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
						return false
					}
				}
				frame.evalStackOffset = len(vm.evaluationStack.Stack)
				vm.callStack.Push(frame)
				vm.pc = int(returnAddress.Int64())
			}

		case CallExt:
			transactionAddress, errArg1 := vm.fetchMany(opCode.Name, 32) // Addresses are 32 bytes (var name: transactionAddress)
			functionHash, errArg2 := vm.fetchMany(opCode.Name, 4)        // Function hash identifies function in external smart contract, first 4 byte of SHA3 hash (var name: functionHash)
			argsToLoad, errArg3 := vm.fetch(opCode.Name)                 // Shows how many arguments to pop from stack and pass to external function (var name: argsToLoad)

			if !vm.checkErrors(opCode.Name, errArg1, errArg2, errArg3) {
				return false
			}

			fmt.Sprint("CALLEXT", transactionAddress, functionHash, argsToLoad)
			//TODO: Invoke new transaction with function hash and arguments, waiting for integration in bazo blockchain to finish

		case Ret:
			callstackTos, err := vm.callStack.Peek()

			if !vm.checkErrors(opCode.Name, err) {
				_ = vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			if (vm.evaluationStack.GetLength() - callstackTos.evalStackOffset) != callstackTos.nrOfReturnTypes {
				_ = vm.evaluationStack.Push([]byte(opCode.Name + ": Number of returned elements does not match."))
				return false
			}

			vm.callStack.Pop()
			vm.pc = callstackTos.returnAddress

		case Size:
			element, err := vm.PopBytes(opCode)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			size := UInt64ToByteArray(uint64(len(element)))

			err = vm.evaluationStack.Push(size)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case StoreSt:
			index, errArgs := vm.fetch(opCode.Name)
			value, errStack := vm.PopBytes(opCode)
			if !vm.checkErrors(opCode.Name, errArgs, errStack) {
				return false
			}

			err = vm.context.SetContractVariable(int(index), value)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case StoreLoc:
			address, errArgs := vm.fetch(opCode.Name)
			right, errStack := vm.PopBytes(opCode)

			if !vm.checkErrors(opCode.Name, errArgs, errStack) {
				return false
			}

			callstackTos, err := vm.callStack.Peek()

			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			callstackTos.variables[int(address)] = right

		case LoadSt:
			index, err := vm.fetch(opCode.Name)
			if !vm.checkErrors(opCode.Name, err) {
				return false
			}

			value, err := vm.context.GetContractVariable(int(index))
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			err = vm.evaluationStack.Push(value)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case LoadLoc:
			address, errArg := vm.fetch(opCode.Name)
			callstackTos, errCallStack := vm.callStack.Peek()

			if !vm.checkErrors(opCode.Name, errArg, errCallStack) {
				return false
			}

			val := callstackTos.variables[int(address)]

			err := vm.evaluationStack.Push(val)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case Address:
			address := vm.context.GetAddress()
			err := vm.evaluationStack.Push(address[:])

			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case Issuer:
			issuer := vm.context.GetIssuer()
			err := vm.evaluationStack.Push(issuer[:])

			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case Balance:
			balance := make([]byte, 8)
			binary.LittleEndian.PutUint64(balance, vm.context.GetBalance())

			err := vm.evaluationStack.Push(balance)

			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case Caller:
			caller := vm.context.GetSender()
			err := vm.evaluationStack.Push(caller[:])

			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case CallVal:
			value := make([]byte, 8)
			binary.LittleEndian.PutUint64(value, vm.context.GetAmount())

			err := vm.evaluationStack.Push(value[:])

			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case CallData:
			td := vm.context.GetTransactionData()
			for i := 0; i < len(td); i++ {
				length := int(td[i]) // Length of parameters

				// Check if Length of TransactionData - the already read data is greater then or equal to the given
				// length parameter
				if len(td)-i-1 < length {
					vm.evaluationStack.Push([]byte(opCode.Name + ": Index out of bounds"))
					return false
				}

				err := vm.evaluationStack.Push(td[i+1 : i+length+1])

				if err != nil {
					vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
					return false
				}

				i += int(td[i]) // Increase to next parameter length
			}

		case NewMap:
			m := CreateMap()

			err = vm.evaluationStack.Push(m)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case MapHasKey:
			mba, err := vm.PopBytes(opCode)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			m, err := MapFromByteArray(mba)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			k, err := vm.PopBytes(opCode)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			result, err := m.MapContainsKey(k)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			vm.evaluationStack.Push(BoolToByteArray(result))

		case MapGetVal:
			mapAsByteArray, err := vm.PopBytes(opCode)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			k, err := vm.PopBytes(opCode)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			m, err := MapFromByteArray(mapAsByteArray)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			v, err := m.GetVal(k)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			err = vm.evaluationStack.Push(v)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case MapSetVal:
			mapAsByteArray, err := vm.PopBytes(opCode)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			m, err := MapFromByteArray(mapAsByteArray)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			k, err := vm.PopBytes(opCode)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			v, err := vm.PopBytes(opCode)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			hasKey, err := m.MapContainsKey(k)
			if err != nil {
				vm.pushError(opCode, err)
				return false
			}

			if hasKey {
				err = m.SetVal(k, v)
			} else {
				err = m.Append(k, v)
			}

			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			err = vm.evaluationStack.Push(m)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case MapRemove:
			mapAsByteArray, err := vm.PopBytes(opCode)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			k, err := vm.PopBytes(opCode)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			m, err := MapFromByteArray(mapAsByteArray)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			err = m.Remove(k)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			err = vm.evaluationStack.Push(m)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case NewArr:
			length, err := vm.PopUnsignedBigInt(opCode)

			if err != nil {
				_ = vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			a := NewArray()

			for i := big.NewInt(0); i.Cmp(&length) == -1; i.Add(i, big.NewInt(1)) {
				err := a.Append([]byte{0})
				if err != nil {
					_ = vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
					return false
				}
			}

			err = vm.evaluationStack.Push(a)
			if err != nil {
				_ = vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}
		case ArrAppend:
			a, aerr := vm.PopBytes(opCode)
			v, verr := vm.PopBytes(opCode)
			if !vm.checkErrors(opCode.Name, verr, aerr) {
				return false
			}

			arr, err := ArrayFromByteArray(a)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			err = arr.Append(v)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": Invalid argument size of ARRAPPEND"))
				return false
			}

			err = vm.evaluationStack.Push(arr)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case ArrInsert:
			a, err := vm.PopBytes(opCode)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			i, err := vm.PopUnsignedBigInt(opCode)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			element, err := vm.PopBytes(opCode)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			arr, err := ArrayFromByteArray(a)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			index, err := BigIntToUInt16(i)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			size, err := arr.GetSize()
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			if index >= size {
				vm.evaluationStack.Push([]byte(opCode.Name + ": Index out of bounds"))
				return false
			}

			err = arr.Insert(index, element)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			err = vm.evaluationStack.Push(arr)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case ArrRemove:
			a, err := vm.PopBytes(opCode)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			i, err := vm.PopUnsignedBigInt(opCode)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			index, err := BigIntToUInt16(i)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			arr, err := ArrayFromByteArray(a)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			err = arr.Remove(index)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			err = vm.evaluationStack.Push(arr)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case ArrAt:
			a, err := vm.PopBytes(opCode)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			i, err := vm.PopUnsignedBigInt(opCode)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			index, err := BigIntToUInt16(i)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			arr, err := ArrayFromByteArray(a)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			element, err := arr.At(index)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			err = vm.evaluationStack.Push(element)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}
		case ArrLen:
			a, err := vm.PopBytes(opCode)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			arr, err := ArrayFromByteArray(a)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			length, err := arr.GetSize()
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}
			lengthBigInt := UInt16ToBigInt(length)
			lengthBytes := BigIntToByteArray(lengthBigInt)

			err = vm.evaluationStack.Push(lengthBytes)

			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}
		case NewStr:
			sizeBytes, err := vm.fetchMany(opCode.Name, 2)
			if err != nil {
				vm.pushError(opCode, err)
				return false
			}

			size, err := ByteArrayToUI16(sizeBytes)
			if err != nil {
				vm.pushError(opCode, err)
				return false
			}

			str := newStruct(size)
			err = vm.evaluationStack.Push(str)
			if err != nil {
				return false
			}
		case StoreFld:
			indexBytes, indexErr := vm.fetchMany(opCode.Name, 2)
			element, elementErr := vm.PopBytes(opCode)
			structBytes, structErr := vm.PopBytes(opCode)

			if !vm.checkErrors(opCode.Name, structErr, indexErr, elementErr) {
				return false
			}

			str, structErr := structFromByteArray(structBytes)
			index, indexErr := ByteArrayToUI16(indexBytes)
			if !vm.checkErrors(opCode.Name, structErr, indexErr) {
				return false
			}

			err := str.storeField(index, element)
			if err != nil {
				vm.pushError(opCode, err)
				return false
			}
			err = vm.evaluationStack.Push(str)
			if err != nil {
				return false
			}
		case LoadFld:
			indexBytes, indexErr := vm.fetchMany(opCode.Name, 2)
			structBytes, structErr := vm.PopBytes(opCode)

			if !vm.checkErrors(opCode.Name, structErr, indexErr) {
				return false
			}

			str, structErr := structFromByteArray(structBytes)
			index, indexErr := ByteArrayToUI16(indexBytes)
			if !vm.checkErrors(opCode.Name, structErr, indexErr) {
				return false
			}

			element, err := str.loadField(index)
			if err != nil {
				vm.pushError(opCode, err)
				return false
			}
			err = vm.evaluationStack.Push(element)
			if err != nil {
				return false
			}
		case SHA3:
			right, err := vm.PopBytes(opCode)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

			hasher := sha3.New256()
			hasher.Write(right)
			hash := hasher.Sum(nil)

			err = vm.evaluationStack.Push(hash)
			if err != nil {
				vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
				return false
			}

		case CheckSig:
			publicKeySig, errArg1 := vm.PopBytes(opCode)
			hash, errArg2 := vm.PopBytes(opCode)

			if !vm.checkErrors(opCode.Name, errArg1, errArg2) {
				return false
			}

			if len(publicKeySig) != 64 {
				vm.evaluationStack.Push([]byte(opCode.Name + ": Not a valid address"))
				return false
			}

			if len(hash) != 32 {
				vm.evaluationStack.Push([]byte(opCode.Name + ": Not a valid hash"))
				return false
			}

			pubKey1Sig1, pubKey2Sig1 := new(big.Int), new(big.Int)
			r, s := new(big.Int), new(big.Int)

			pubKey1Sig1.SetBytes(publicKeySig[:32])
			pubKey2Sig1.SetBytes(publicKeySig[32:])

			sig1 := vm.context.GetSig1()
			r.SetBytes(sig1[:32])
			s.SetBytes(sig1[32:])

			pubKey := ecdsa.PublicKey{elliptic.P256(), pubKey1Sig1, pubKey2Sig1}

			result := ecdsa.Verify(&pubKey, hash, r, s)
			vm.evaluationStack.Push(BoolToByteArray(result))

		case ErrHalt:
			return false

		case Halt:
			return true
		}
	}
}

func (vm *VM) fetch(errorLocation string) (element byte, err error) {
	tempPc := vm.pc
	if len(vm.code) > tempPc {
		vm.pc++
		return vm.code[tempPc], nil
	}
	return 0, errors.New("Instruction set out of bounds")
}

func (vm *VM) fetchMany(errorLocation string, argument int) (elements []byte, err error) {
	tempPc := vm.pc
	if len(vm.code)-tempPc > argument {
		vm.pc += argument
		return vm.code[tempPc : tempPc+argument], nil
	}
	return []byte{}, errors.New("Instruction set out of bounds")
}

func (vm *VM) checkErrors(errorLocation string, errors ...error) bool {
	for i, err := range errors {
		if err != nil {
			vm.evaluationStack.Push([]byte(errorLocation + ": " + errors[i].Error()))
			return false
		}
	}
	return true
}

func (vm *VM) pushError(opCode OpCode, err error) {
	_ = vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
}

// PopBytes pops bytes from the evaluation stack.
func (vm *VM) PopBytes(opCode OpCode) (elements []byte, err error) {
	bytes, err := vm.evaluationStack.Pop()
	if err != nil {
		return nil, err
	}

	elementSize := (len(bytes) + 64 - 1) / 64

	gasCost := opCode.gasFactor * uint64(elementSize)
	if int64(vm.fee-gasCost) < 0 {
		return nil, errors.New("Out of gas")
	}

	vm.fee -= gasCost

	return bytes, nil
}

// PopSignedBigInt pops bytes from evaluation stack and convert it to a big integer with sign.
func (vm *VM) PopSignedBigInt(opCode OpCode) (bigInt big.Int, err error) {
	bytes, err := vm.evaluationStack.Pop()
	if err != nil {
		return *big.NewInt(0), err
	}

	elementSize := (len(bytes) + 64 - 1) / 64

	gasCost := opCode.gasFactor * uint64(elementSize)
	if int64(vm.fee-gasCost) < 0 {
		return *big.NewInt(0), errors.New("Out of gas")
	}

	vm.fee -= gasCost

	result, err := SignedBigIntConversion(bytes, err)
	return result, err
}

// PopUnsignedBigInt pops bytes from evaluation stack and convert it to an unsigned big integer.
func (vm *VM) PopUnsignedBigInt(opCode OpCode) (bigInt big.Int, err error) {
	bytes, err := vm.evaluationStack.Pop()
	if err != nil {
		return *big.NewInt(0), err
	}

	elementSize := (len(bytes) + 64 - 1) / 64

	gasCost := opCode.gasFactor * uint64(elementSize)
	if int64(vm.fee-gasCost) < 0 {
		return *big.NewInt(0), errors.New("Out of gas")
	}

	vm.fee -= gasCost

	result, err := UnsignedBigIntConversion(bytes, err)
	return result, err
}

// PeekResult returns the element on top of the stack
func (vm *VM) PeekResult() (element []byte, err error) {
	return vm.evaluationStack.PeekBytes()
}

// PeekEvalStack returns a copy of the complete evaluation stack
func (vm *VM) PeekEvalStack() [][]byte {
	evalStack := vm.evaluationStack.Stack
	copiedStack := make([][]byte, len(evalStack))

	for i := range evalStack {
		copiedStack[i] = make([]byte, len(evalStack[i]))
		copy(copiedStack[i], evalStack[i])
	}
	return copiedStack
}

// GetErrorMsg peeks bytes from evaluation stack and returns the error message.
func (vm *VM) GetErrorMsg() string {
	tos, err := vm.evaluationStack.PeekBytes()
	if err != nil {
		return "Peek on empty Stack"
	}
	return string(tos)
}

type bigIntAction func(left *big.Int, right *big.Int)

func (vm *VM) evaluateBigIntOperation(opCode OpCode, exec bigIntAction) bool {
	right, rerr := vm.PopSignedBigInt(opCode)
	left, lerr := vm.PopSignedBigInt(opCode)

	if !vm.checkErrors(opCode.Name, rerr, lerr) {
		return false
	}

	exec(&left, &right)
	err := vm.evaluationStack.Push(SignedByteArrayConversion(left))

	if err != nil {
		_ = vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
		return false
	}
	return true
}

func (vm *VM) evaluateRelationalComp(opCode OpCode, expectedResult ...int) bool {
	right, rerr := vm.PopBytes(opCode)
	left, lerr := vm.PopBytes(opCode)
	if !vm.checkErrors(opCode.Name, rerr, lerr) {
		return false
	}

	var result int
	// char has always one byte
	if len(left) == 1 && len(right) == 1 {
		result = bytes.Compare(left, right)
	} else {
		leftInt, lerr := SignedBigIntConversion(left, nil)
		rightInt, rerr := SignedBigIntConversion(right, nil)

		if !vm.checkErrors(opCode.Name, rerr, lerr) {
			return false
		}
		result = leftInt.Cmp(&rightInt)
	}

	var compResult bool
	for _, r := range expectedResult {
		if r == result {
			compResult = true
		}
	}

	err := vm.evaluationStack.Push(BoolToByteArray(compResult))
	if err != nil {
		_ = vm.evaluationStack.Push([]byte(opCode.Name + ": " + err.Error()))
		return false
	}
	return true
}
