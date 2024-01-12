package compiler

import (
	"fmt"
	"encoding/binary"
	"teriyake/goo/parser"
)

type Instruction struct {
	Opcode   Opcode
	Operands []int
}

type Opcode int

const (
	ADD Opcode = iota
	SUB
	GRT
	IF
	PRINT
	PUSH_VARIABLE Opcode = iota
	PUSH_NUMBER
	PUSH_STRING
)

type BytecodeInstruction struct {
	Opcode   Opcode
	Operands []interface{}
}

type Compiler struct {
	bytecode []byte
}

func NewCompiler() *Compiler {
	return &Compiler{}
}

func (c *Compiler) CompileASTByte(ast interface{}) ([]byte, error) {
	c.bytecode = []byte{}

	err := c.compileNode(ast)
	if err != nil {
		return nil, err
	}

	return c.bytecode, nil
}

func (c *Compiler) CompileAST(ast interface{}) ([]BytecodeInstruction,  error) {
	c.bytecode = []byte{}

	err := c.compileNode(ast)
	if err != nil {
		return nil, err
	}

	bytecodeInstructions, err := convertBytecode(c.bytecode)
	if err != nil {
		return nil, err
	}

	return bytecodeInstructions, nil
}

func (c *Compiler) compileNode(node interface{}) error {
	switch n := node.(type) {
	case []interface{}:
		if len(n) == 0 {
			return fmt.Errorf("empty expression")
		}

		for _, operand := range n[1:] {
			err := c.compileNode(operand)
			if err != nil {
				return err
			}
		}

		if operatorNode, ok := n[0].(parser.Operator); ok {
			switch operatorNode.Value {
			case "+":
				c.emit(ADD)
			case "-":
				c.emit(SUB)
			case ">":
				c.emit(GRT)
			// Add more cases for other operators
			default:
				return fmt.Errorf("unknown operator: %s", operatorNode.Value)
			}
		} else {
			return c.compileNode(n[0])
		}

	case parser.Identifier:
		fmt.Printf("Emitting Identifier: %v\n", n.Value)
		c.emit(PUSH_VARIABLE, n.Value)

	case parser.Number:
		fmt.Printf("Emitting Number: %v\n", n.Value)
		c.emit(PUSH_NUMBER, n.Value)

	case parser.String:
		fmt.Printf("Emitting String: %v\n", n.Value)
		c.emit(PUSH_STRING, n.Value)

	default:
		return fmt.Errorf("unknown node type: %T", n)
	}

	return nil
}

func (c *Compiler) emit(opcode Opcode, operands ...interface{}) {
	opcodeBytes := []byte{byte(opcode)}
	operandBytes := serializeOperands(operands)
	c.bytecode = append(c.bytecode, opcodeBytes...)
	c.bytecode = append(c.bytecode, operandBytes...)
}

func serializeOperands(operands []interface{}) []byte {
	var result []byte

	for _, operand := range operands {
		fmt.Printf("Operand: %v, Type: %T\n", operand, operand)

		switch v := operand.(type) {
		case int:
			buf := make([]byte, 4)
			binary.LittleEndian.PutUint32(buf, uint32(v))
			result = append(result, buf...)

		case string:
			strBytes := []byte(v)
			if len(strBytes) > 1024 {
				fmt.Println("String operand too long")
				continue
			}

			lengthBuf := make([]byte, 4) // 4 bytes to store the length of the string
			binary.LittleEndian.PutUint32(lengthBuf, uint32(len(strBytes)))
			result = append(result, lengthBuf...)
			result = append(result, strBytes...)

		default:
			fmt.Printf("Unsupported operand type: %T\n", v)
		}
	}

	return result
}

func convertBytecode(rawBytecode []byte) ([]BytecodeInstruction, error) {
	var instructions []BytecodeInstruction
	i := 0

	for i < len(rawBytecode) {
		opcode := Opcode(rawBytecode[i])
		i++

		var operands []interface{}
		switch opcode {
		case PUSH_NUMBER:
			if i+4 > len(rawBytecode) {
				return nil, fmt.Errorf("invalid bytecode, unexpected end of data")
			}
			number := int(binary.LittleEndian.Uint32(rawBytecode[i : i+4]))
			operands = append(operands, number)
			i += 4

		case PUSH_STRING:
			if i+4 > len(rawBytecode) {
				return nil, fmt.Errorf("invalid bytecode, unexpected end of data")
			}
			strLen := int(binary.LittleEndian.Uint32(rawBytecode[i : i+4]))
			i += 4

			if i+strLen > len(rawBytecode) {
				return nil, fmt.Errorf("invalid bytecode, unexpected end of data")
			}
			str := string(rawBytecode[i : i+strLen])
			operands = append(operands, str)
			i += strLen

		// Add cases for other opcodes that have operands
		// ...

		default:
			// Opcodes without operands do not modify 'i'
		}

		instructions = append(instructions, BytecodeInstruction{Opcode: opcode, Operands: operands})
	}

	return instructions, nil
}
