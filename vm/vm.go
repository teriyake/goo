package vm

import (
	"encoding/binary"
	"fmt"
	"math"
	"teriyake/goo/compiler"
)

type RuntimeSymbolTable struct {
	symbols map[string]interface{}
	parent  *RuntimeSymbolTable
}

func NewRuntimeSymbolTable(parent *RuntimeSymbolTable) *RuntimeSymbolTable {
	return &RuntimeSymbolTable{
		symbols: make(map[string]interface{}),
		parent:  parent,
	}
}

func (rst *RuntimeSymbolTable) Set(name string, value interface{}) {
	rst.symbols[name] = value
}

func (rst *RuntimeSymbolTable) Get(name string) (interface{}, bool) {
	value, exists := rst.symbols[name]
	if !exists && rst.parent != nil {
		value, exists = rst.parent.Get(name)
	}
	return value, exists
}

type FunctionMetadata struct {
	StartAddress int
	ParamCount   int
	ParamNames   []string
}

type CallStackEntry struct {
	returnAddress int
	symbolTable   *RuntimeSymbolTable
}

type VM struct {
	stack []interface{}
	pc    int
	code  []compiler.BytecodeInstruction
	//symbolTable map[string]interface{}
	symbolTableStack []*RuntimeSymbolTable
	functions        map[string]FunctionMetadata
	callStack        []CallStackEntry
}

func NewVM(code []compiler.BytecodeInstruction) *VM {
	globalSymbolTable := NewRuntimeSymbolTable(nil)
	return &VM{
		stack:            make([]interface{}, 0),
		pc:               0,
		code:             code,
		symbolTableStack: []*RuntimeSymbolTable{globalSymbolTable},
		functions:        make(map[string]FunctionMetadata),
		callStack:        make([]CallStackEntry, 0),
	}
}

func (vm *VM) ResolveParamName(funcName string, index int) (string, error) {
	funcMetadata, exists := vm.functions[funcName]
	if !exists {
		return "", fmt.Errorf("function %s not found", funcName)
	}

	if index < 0 || index >= len(funcMetadata.ParamNames) {
		return "", fmt.Errorf("parameter index out of range for function %s", funcName)
	}

	return funcMetadata.ParamNames[index], nil
}

func (vm *VM) push(value interface{}) {
	vm.stack = append(vm.stack, value)
}

func (vm *VM) pop() (interface{}, error) {
	if len(vm.stack) == 0 {
		return nil, fmt.Errorf("stack underflow: cannot pop from an empty stack")
	}

	topIndex := len(vm.stack) - 1
	topElement := vm.stack[topIndex]

	vm.stack = vm.stack[:topIndex]

	return topElement, nil
}

func (vm *VM) Run() error {
	for vm.pc < len(vm.code) {
		instruction := vm.code[vm.pc]
		fmt.Printf("Executing Instruction at PC %v: Opcode %d, Operands %v\n", vm.pc, instruction.Opcode, instruction.Operands)
		vm.pc++

		switch instruction.Opcode {
		case compiler.PUSH_NUMBER:
			if len(instruction.Operands) < 1 {
				return fmt.Errorf("PUSH_NUMBER instruction requires an operand")
			}
			operandBytes, ok := instruction.Operands[0].([]byte)
			if !ok || len(operandBytes) != 8 {
				return fmt.Errorf("Invalid operand for PUSH_NUMBER instruction")
			}
			bits := binary.LittleEndian.Uint64(operandBytes)
			floatValue := math.Float64frombits(bits)
			vm.stack = append(vm.stack, floatValue)
			fmt.Printf("Stack after PUSH_NUMBER: %v\n", vm.stack)
		case compiler.PUSH_BOOL:
			if len(instruction.Operands) < 1 {
				return fmt.Errorf("PUSH_BOOL instruction requires an operand")
			}
			operandByte, ok := instruction.Operands[0].(byte)
			if !ok {
				return fmt.Errorf("Invalid operand for PUSH_BOOL instruction")
			}
			boolValue := operandByte != 0
			vm.stack = append(vm.stack, boolValue)
			fmt.Printf("Stack after PUSH_BOOL: %v\n", vm.stack)

		case compiler.PUSH_STRING:
			if len(instruction.Operands) < 1 {
				return fmt.Errorf("PUSH_STRING instruction requires an operand")
			}
			operandBytes, ok := instruction.Operands[0].([]byte)
			if !ok {
				return fmt.Errorf("Invalid operand type for PUSH_STRING instruction")
			}
			stringValue := string(operandBytes)
			vm.stack = append(vm.stack, stringValue)
			fmt.Printf("Stack after PUSH_STRING: %v\n", vm.stack)
		case compiler.ADD:
			if len(vm.stack) < 2 {
				return fmt.Errorf("ADD instruction requires at least 2 values on the stack")
			}
			operand2, ok1 := vm.stack[len(vm.stack)-1].(float64)
			operand1, ok2 := vm.stack[len(vm.stack)-2].(float64)
			if !ok1 || !ok2 {
				return fmt.Errorf("ADD instruction requires float operands")
			}
			result := operand1 + operand2
			vm.stack = vm.stack[:len(vm.stack)-2]
			vm.stack = append(vm.stack, result)
			fmt.Printf("Stack after ADD: %v\n", vm.stack)
		case compiler.SUB:
			if len(vm.stack) < 2 {
				return fmt.Errorf("SUB instruction requires at least 2 values on the stack")
			}
			operand2, ok1 := vm.stack[len(vm.stack)-1].(float64)
			operand1, ok2 := vm.stack[len(vm.stack)-2].(float64)
			if !ok1 || !ok2 {
				return fmt.Errorf("SUB instruction requires float operands")
			}
			result := operand1 - operand2
			vm.stack = vm.stack[:len(vm.stack)-2]
			vm.stack = append(vm.stack, result)
			fmt.Printf("Stack after SUB: %v\n", vm.stack)
		case compiler.GRT:
			if len(vm.stack) < 2 {
				return fmt.Errorf("GRT instruction requires at least 2 values on the stack")
			}
			operand2, ok1 := vm.stack[len(vm.stack)-1].(float64)
			operand1, ok2 := vm.stack[len(vm.stack)-2].(float64)
			if !ok1 || !ok2 {
				return fmt.Errorf("GRT instruction requires float operands")
			}
			result := operand1 > operand2
			vm.stack = vm.stack[:len(vm.stack)-2]
			vm.stack = append(vm.stack, result)
			fmt.Printf("Stack after GRT: %v\n", vm.stack)
		case compiler.LESS:
			if len(vm.stack) < 2 {
				return fmt.Errorf("LESS instruction requires at least 2 values on the stack")
			}
			operand2, ok1 := vm.stack[len(vm.stack)-1].(float64)
			operand1, ok2 := vm.stack[len(vm.stack)-2].(float64)
			if !ok1 || !ok2 {
				return fmt.Errorf("LESS instruction requires float operands")
			}
			result := operand1 < operand2
			vm.stack = vm.stack[:len(vm.stack)-2]
			vm.stack = append(vm.stack, result)
			fmt.Printf("Stack after LESS: %v\n", vm.stack)

		case compiler.EQ:
			if len(vm.stack) < 2 {
				return fmt.Errorf("EQ instruction requires at least 2 values on the stack")
			}

			operand2 := vm.stack[len(vm.stack)-1]
			operand1 := vm.stack[len(vm.stack)-2]

			if _, ok1 := operand1.(float64); ok1 {
				if _, ok2 := operand2.(float64); ok2 {
					result := operand1.(float64) == operand2.(float64)
					vm.stack = vm.stack[:len(vm.stack)-2]
					vm.stack = append(vm.stack, result)
					fmt.Printf("Stack after EQ: %v\n", vm.stack)
				} else {
					return fmt.Errorf("EQ instruction requires operands of the same type")
				}
			} else if _, ok1 := operand1.(string); ok1 {
				if _, ok2 := operand2.(string); ok2 {
					result := operand1.(string) == operand2.(string)
					vm.stack = vm.stack[:len(vm.stack)-2]
					vm.stack = append(vm.stack, result)
					fmt.Printf("Stack after EQ: %v\n", vm.stack)
				} else {
					return fmt.Errorf("EQ instruction requires operands of the same type")
				}
			} else {
				return fmt.Errorf("EQ instruction requires operands of the same type")
			}

		case compiler.NEQ:
			if len(vm.stack) < 2 {
				return fmt.Errorf("NEQ instruction requires at least 2 values on the stack")
			}

			operand2 := vm.stack[len(vm.stack)-1]
			operand1 := vm.stack[len(vm.stack)-2]

			if _, ok1 := operand1.(float64); ok1 {
				if _, ok2 := operand2.(float64); ok2 {
					result := operand1.(float64) != operand2.(float64)
					vm.stack = vm.stack[:len(vm.stack)-2]
					vm.stack = append(vm.stack, result)
					fmt.Printf("Stack after NEQ: %v\n", vm.stack)
				} else {
					return fmt.Errorf("NEQ instruction requires operands of the same type")
				}
			} else if _, ok1 := operand1.(string); ok1 {
				if _, ok2 := operand2.(string); ok2 {
					result := operand1.(string) != operand2.(string)
					vm.stack = vm.stack[:len(vm.stack)-2]
					vm.stack = append(vm.stack, result)
					fmt.Printf("Stack after NEQ: %v\n", vm.stack)
				} else {
					return fmt.Errorf("NEQ instruction requires operands of the same type")
				}
			} else {
				return fmt.Errorf("NEQ instruction requires operands of the same type")
			}
		case compiler.PRINT:
			if len(vm.stack) < 1 {
				return fmt.Errorf("PRINT instruction requires a value on the stack")
			}
			value := vm.stack[len(vm.stack)-1]
			fmt.Println(value)
			vm.stack = vm.stack[:len(vm.stack)-1]

		case compiler.DEFINE_VARIABLE:
			if len(instruction.Operands) < 1 {
				return fmt.Errorf("DEFINE_VARIABLE instruction requires a variable name as operand")
			}

			varBytes, ok := instruction.Operands[0].([]byte)
			if !ok {
				return fmt.Errorf("Invalid operand type for DEFINE_VARIABLE instruction")
			}
			varName := string(varBytes)

			if len(vm.stack) == 0 {
				return fmt.Errorf("No value on stack to assign to variable %s", varName)
			}

			value := vm.stack[len(vm.stack)-1]
			vm.stack = vm.stack[:len(vm.stack)-1]

			currentSymbolTable := vm.symbolTableStack[len(vm.symbolTableStack)-1]
			if _, exists := currentSymbolTable.Get(varName); exists {
				return fmt.Errorf("Variable %s is immutable and has already been defined", varName)
			}
			currentSymbolTable.Set(varName, value)

			fmt.Printf("Variable %s defined with value: %v\n", varName, value)

		case compiler.PUSH_VARIABLE:
			varNameBytes, ok := instruction.Operands[0].([]byte)
			if !ok {
				return fmt.Errorf("Invalid operand for PUSH_VARIABLE instruction")
			}
			varName := string(varNameBytes)

			currentSymbolTable := vm.symbolTableStack[len(vm.symbolTableStack)-1]

			value, ok := currentSymbolTable.Get(varName)
			if !ok {
				return fmt.Errorf("Variable %s not defined", varName)
			}

			vm.stack = append(vm.stack, value)
			fmt.Printf("Stack after PUSH_VARIABLE (%s): %v\n", varName, vm.stack)
		case compiler.DEFINE_FUNCTION:
			funcNameBytes, ok := instruction.Operands[0].([]byte)
			if !ok {
				return fmt.Errorf("Invalid operand for function name in DEFINE_FUNCTION instruction")
			}
			funcName := string(funcNameBytes)

			startAddressBytes, ok := instruction.Operands[1].([]byte)
			if !ok || len(startAddressBytes) != 4 {
				return fmt.Errorf("Invalid or missing start address in DEFINE_FUNCTION instruction")
			}
			startAddress := int(binary.LittleEndian.Uint32(startAddressBytes)) + 1

			paramCountBytes, ok := instruction.Operands[2].([]byte)
			if !ok || len(paramCountBytes) != 4 {
				return fmt.Errorf("Invalid or missing parameter count in DEFINE_FUNCTION instruction")
			}
			paramCount := int(binary.LittleEndian.Uint32(paramCountBytes))
			var paramNames []string
			if len(instruction.Operands) > 3 {
				listLenBytes, ok := instruction.Operands[3].([]byte)
				if !ok || len(listLenBytes) != 4 {
					return fmt.Errorf("Invalid or missing length of parameter list in DEFINE_FUNCTION instruction")
				}
				listLen := int(binary.LittleEndian.Uint32(listLenBytes))

				for i := 0; i < listLen; i++ {
					paramNameLenBytes, ok := instruction.Operands[4+i*2].([]byte)
					if !ok || len(paramNameLenBytes) != 4 {
						return fmt.Errorf("Invalid or missing length for parameter name in DEFINE_FUNCTION instruction")
					}
					paramNameLen := int(binary.LittleEndian.Uint32(paramNameLenBytes))

					paramNameBytes, ok := instruction.Operands[5+i*2].([]byte)
					if !ok || len(paramNameBytes) != paramNameLen {
						return fmt.Errorf("Invalid or missing parameter name in DEFINE_FUNCTION instruction")
					}
					paramName := string(paramNameBytes)
					paramNames = append(paramNames, paramName)
				}
			}
			vm.functions[funcName] = FunctionMetadata{
				StartAddress: startAddress,
				ParamCount:   paramCount,
				ParamNames:   paramNames,
			}

			fmt.Printf("Function %s defined with param count: %v and params: %v\n", funcName, paramCount, paramNames)
			fmt.Printf("Function %s starts at: %v\n", funcName, startAddress)

			fmt.Printf("Current PC: %v\n", vm.pc)

		case compiler.CALL_FUNCTION:
			funcNameBytes, ok := instruction.Operands[0].([]byte)
			if !ok {
				return fmt.Errorf("Invalid operand for CALL_FUNCTION instruction")
			}
			funcName := string(funcNameBytes)

			functionMetadata, ok := vm.functions[funcName]
			if !ok {
				return fmt.Errorf("Function %s not defined", funcName)
			}

			vm.callStack = append(vm.callStack, CallStackEntry{
				returnAddress: vm.pc,
				symbolTable:   vm.symbolTableStack[len(vm.symbolTableStack)-1],
			})

			vm.pc = functionMetadata.StartAddress

			newSymbolTable := NewRuntimeSymbolTable(vm.symbolTableStack[len(vm.symbolTableStack)-1])
			vm.symbolTableStack = append(vm.symbolTableStack, newSymbolTable)

			for i, paramName := range functionMetadata.ParamNames {
				if len(vm.stack) < len(functionMetadata.ParamNames)-i {
					return fmt.Errorf("Not enough arguments for function %s", funcName)
				}
				paramValue := vm.stack[len(vm.stack)-len(functionMetadata.ParamNames)+i]
				newSymbolTable.Set(paramName, paramValue)
			}

			vm.stack = vm.stack[:len(vm.stack)-len(functionMetadata.ParamNames)]

			continue
		case compiler.RETURN:
			if len(vm.callStack) == 0 {
				return fmt.Errorf("Call stack is empty on return")
			}

			returnValue, err := vm.pop()
			if err != nil {
				return fmt.Errorf("No return value found on the stack")
			}

			callStackEntry := vm.callStack[len(vm.callStack)-1]
			vm.callStack = vm.callStack[:len(vm.callStack)-1]

			vm.pc = callStackEntry.returnAddress
			vm.symbolTableStack = vm.symbolTableStack[:len(vm.symbolTableStack)-1]
			vm.symbolTableStack = append(vm.symbolTableStack, callStackEntry.symbolTable)

			vm.push(returnValue)

			continue
		case compiler.JUMP:
			if len(instruction.Operands) < 1 {
				return fmt.Errorf("JUMP instruction requires an operand")
			}
			jumpOffsetBytes, ok := instruction.Operands[0].([]byte)
			if !ok || len(jumpOffsetBytes) != 4 {
				return fmt.Errorf("Invalid operand for JUMP instruction")
			}
			jumpOffset := binary.LittleEndian.Uint32(jumpOffsetBytes)

			fmt.Printf("Current PC: %v\tJump offset: %v\n", vm.pc, jumpOffset)
			vm.pc += int(jumpOffset)
			fmt.Printf("Updated PC: %v\n", vm.pc)
			if vm.pc >= len(vm.code) {
				return fmt.Errorf("Jump leads to invalid instruction index")
			}
		case compiler.IF:
			if len(vm.stack) < 1 {
				return fmt.Errorf("IF instruction requires a condition value on the stack")
			}
			condition, ok := vm.stack[len(vm.stack)-1].(bool)
			if !ok {
				return fmt.Errorf("IF instruction requires a boolean condition value on the stack")
			}
			vm.stack = vm.stack[:len(vm.stack)-1]

			if !condition {
				vm.jumpToOpcode(compiler.ELSE)
			}
		case compiler.ELSE:
			vm.jumpToOpcode(compiler.ENDIF)

		case compiler.ENDIF:

		default:
			return fmt.Errorf("Unknown instruction: %v", instruction.Opcode)
		}
	}

	return nil
}

func (vm *VM) jumpToOpcode(opcode compiler.Opcode) {
	for vm.pc < len(vm.code) {
		if vm.code[vm.pc].Opcode == opcode {
			vm.pc++
			break
		}
		vm.pc++
	}
}
