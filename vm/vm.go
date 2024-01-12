package vm

import (
	"fmt"
	"teriyake/goo/compiler"
)

type VM struct {
	stack []interface{}
	pc    int
	code  []compiler.BytecodeInstruction
}

func NewVM(code []compiler.BytecodeInstruction) *VM {
	return &VM{
		stack: make([]interface{}, 0),
		pc:    0,
		code:  code,
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
			vm.stack = append(vm.stack, instruction.Operands[0])
			fmt.Printf("Stack after PUSH_NUMBER: %v\n", vm.stack)
		case compiler.ADD:
			if len(vm.stack) < 2 {
				return fmt.Errorf("ADD instruction requires at least 2 values on the stack")
			}
			operand2, ok1 := vm.stack[len(vm.stack)-1].(int)
			operand1, ok2 := vm.stack[len(vm.stack)-2].(int)
			if !ok1 || !ok2 {
				return fmt.Errorf("ADD instruction requires integer operands")
			}
			result := operand1 + operand2
			vm.stack = vm.stack[:len(vm.stack)-2] 
			vm.stack = append(vm.stack, result)
			fmt.Printf("Stack after ADD: %v\n", vm.stack)
		case compiler.SUB:
			// Handle SUB instruction
			// Implement subtraction similar to ADD
		case compiler.GRT:
			// Handle GRT instruction
			// Implement greater-than comparison
		case compiler.PRINT:
			// Handle PRINT instruction
			if len(vm.stack) < 1 {
				return fmt.Errorf("PRINT instruction requires a value on the stack")
			}
			value := vm.stack[len(vm.stack)-1]
			fmt.Println(value)
			vm.stack = vm.stack[:len(vm.stack)-1]
		// Add cases for other instructions like SUB, GRT, IF, etc.
		default:
			return fmt.Errorf("Unknown instruction: %v", instruction.Opcode)
		}
	}

	return nil
}
