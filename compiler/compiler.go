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
	PUSH_VARIABLE Opcode = iota + 10
	PUSH_NUMBER
	PUSH_STRING
	DEFINE_VARIABLE Opcode = iota + 20
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
	fmt.Println("Entering compileNode with node:", node)
    
	switch n := node.(type) {
    case []interface{}:
        if len(n) == 0 {
            return fmt.Errorf("empty expression")
        }

		if identifierNode, ok := n[0].(parser.Identifier); ok {
            switch identifierNode.Value {
            case "def":
				fmt.Println("Handling 'def' statement")
                if len(n) != 3 {
                    return fmt.Errorf("def expects two arguments")
                }
                varName, ok := n[1].(parser.Identifier)
                if !ok {
                    return fmt.Errorf("expected a variable name as the second argument to def")
                }
				fmt.Printf("Variable name for 'def': %s\n", varName.Value)
                
				err := c.compileNode(n[2]) // Compile the value to be assigned
                if err != nil {
                    return err
                }
				fmt.Printf("Emitting DEFINE_VARIABLE for %s\n", varName.Value)
                c.emit(DEFINE_VARIABLE, varName.Value)
                return nil

            case "print":
                if len(n) != 2 {
                    return fmt.Errorf("print expects one argument")
                }
                err := c.compileNode(n[1]) // Compile the argument to print
                if err != nil {
                    return err
                }
                c.emit(PRINT)
                return nil

            // ... handle other specific keywords ...

            }
        }

		for _, operand := range n {
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
            // ... other operators ...
            default:
                return fmt.Errorf("unknown operator: %s", operatorNode.Value)
            }
		}

    case parser.Identifier:
        // Handle identifier (variable) nodes
        fmt.Printf("Emitting Identifier: %v\n", n.Value)
        c.emit(PUSH_VARIABLE, n.Value)

    case parser.Number:
        // Handle number nodes
        fmt.Printf("Emitting Number: %v\n", n.Value)
        c.emit(PUSH_NUMBER, n.Value)

    case parser.String:
        // Handle string nodes
        fmt.Printf("Emitting String: %v\n", n.Value)
        c.emit(PUSH_STRING, n.Value)
	
    default:
        return fmt.Errorf("unknown node type: %T", n)
    }

	fmt.Println("Exiting compileNode")
    return nil
}

func (c *Compiler) compileNode2(node interface{}) error {
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
	fmt.Printf("Emitting opcode: %d with operands: %v\n", opcode, operands)
	opcodeBytes := []byte{byte(opcode)}
	operandBytes := serializeOperands(operands)
	c.bytecode = append(c.bytecode, opcodeBytes...)
	c.bytecode = append(c.bytecode, operandBytes...)
}

func serializeOperands(operands []interface{}) []byte {
	var result []byte

	for _, operand := range operands {
		fmt.Printf("Serializing Operand: %v, Type: %T\n", operand, operand)

		switch v := operand.(type) {
		case int:
			buf := make([]byte, 4)
			binary.LittleEndian.PutUint32(buf, uint32(v))
			result = append(result, buf...)
		case string:
            fmt.Printf("Serializing string operand: %s\n", v)
            strBytes := []byte(v)
            lengthBuf := make([]byte, 4)
            binary.LittleEndian.PutUint32(lengthBuf, uint32(len(strBytes)))
			fmt.Println(lengthBuf)
			fmt.Println(strBytes)
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
       case PUSH_VARIABLE:
            if i+4 > len(rawBytecode) {
                return nil, fmt.Errorf("invalid bytecode, unexpected end of data")
            }
            varNameLen := int(binary.LittleEndian.Uint32(rawBytecode[i : i+4]))
            i += 4

            if i+varNameLen > len(rawBytecode) {
                return nil, fmt.Errorf("invalid bytecode, unexpected end of data")
            }
            varName := string(rawBytecode[i : i+varNameLen])
            operands = append(operands, varName)
            i += varNameLen

        case DEFINE_VARIABLE:
            if i+4 > len(rawBytecode) {
                return nil, fmt.Errorf("invalid bytecode, unexpected end of data")
            }
            varNameLen := int(binary.LittleEndian.Uint32(rawBytecode[i : i+4]))
            i += 4

            if i+varNameLen > len(rawBytecode) {
                return nil, fmt.Errorf("invalid bytecode, unexpected end of data")
            }
            varName := string(rawBytecode[i : i+varNameLen])
            operands = append(operands, varName)
            i += varNameLen

		default:
			// Opcodes without operands do not modify 'i'
		}

		instructions = append(instructions, BytecodeInstruction{Opcode: opcode, Operands: operands})
	}

	return instructions, nil
}
