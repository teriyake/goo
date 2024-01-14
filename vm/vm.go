package vm

import (
	"encoding/binary"
	"fmt"
	"math"
	"teriyake/goo/compiler"
)

type VM struct {
	stack       []interface{}
	pc          int
	code        []compiler.BytecodeInstruction
	symbolTable map[string]interface{}
}

func NewVM(code []compiler.BytecodeInstruction) *VM {
	return &VM{
		stack:       make([]interface{}, 0),
		pc:          0,
		code:        code,
		symbolTable: make(map[string]interface{}),
	}
}

func (vm *VM) Run() error {
	for vm.pc < len(vm.code) {
		instruction := vm.code[vm.pc]
		fmt.Printf("Executing Instruction: Opcode %d, Operands %v\n", instruction.Opcode, instruction.Operands)
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
			if !ok || len(vm.stack) == 0 {
				return fmt.Errorf("DEFINE_VARIABLE requires a variable name and a value on the stack")
			}
			varName := string(varBytes)
			vm.symbolTable[varName] = vm.stack[len(vm.stack)-1]
			vm.stack = vm.stack[:len(vm.stack)-1]
			fmt.Printf("Variable %s defined with value: %v\n", varName, vm.symbolTable[varName])
		case compiler.PUSH_VARIABLE:
			if len(instruction.Operands) < 1 {
				return fmt.Errorf("PUSH_VARIABLE instruction requires a variable name as operand")
			}
			varBytes, ok := instruction.Operands[0].([]byte)
			if !ok {
				return fmt.Errorf("Invalid operand type for PUSH_VARIABLE instruction")
			}
			varName := string(varBytes)
			value, ok := vm.symbolTable[varName]
			if !ok {
				return fmt.Errorf("Variable %s not defined", varName)
			}
			vm.stack = append(vm.stack, value)
			fmt.Printf("Stack after PUSH_VARIABLE (%s): %v\n", varName, vm.stack)
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
				// Jump to the ELSE or ENDIF
				for vm.pc < len(vm.code) {
					nextInstruction := vm.code[vm.pc]
					if nextInstruction.Opcode == compiler.ELSE || nextInstruction.Opcode == compiler.ENDIF {
						break
					}
					vm.pc++
				}
			}
		case compiler.ELSE:
			// Skip the "else" block by jumping to the ENDIF
			for vm.pc < len(vm.code) {
				nextInstruction := vm.code[vm.pc]
				if nextInstruction.Opcode == compiler.ENDIF {
					break
				}
				vm.pc++
			}

		case compiler.ENDIF:
			// No action needed for ENDIF
		default:
			return fmt.Errorf("Unknown instruction: %v", instruction.Opcode)
		}
	}

	return nil
}
