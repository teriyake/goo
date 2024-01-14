package compiler

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
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
	LESS
	EQ
	NEQ
	IF
	ELSE
	ENDIF
	PRINT
	PUSH_VARIABLE Opcode = iota + 20
	PUSH_NUMBER
	PUSH_BOOL
	PUSH_STRING
	DEFINE_VARIABLE Opcode = iota + 30
	DEFINE_FUNCTION
	RETURN
	JUMP
	CALL_FUNCTION
)

type BytecodeInstruction struct {
	Opcode   Opcode
	Operands []interface{}
}

type SymbolType int

const (
	VariableSymbol SymbolType = iota
	FunctionSymbol
)

type Symbol struct {
	Name       string
	Type       SymbolType
	ParamNames []string
}

type SymbolTable struct {
	Symbols map[string]Symbol
	Parent  *SymbolTable
}

func NewSymbolTable(parent *SymbolTable) *SymbolTable {
	return &SymbolTable{
		Symbols: make(map[string]Symbol),
		Parent:  parent,
	}
}

func (st *SymbolTable) DefineSymbol(name string, symbolType SymbolType) {
	st.Symbols[name] = Symbol{
		Name: name,
		Type: symbolType,
	}
}

func (st *SymbolTable) DefineVariable(name string) {
	st.DefineSymbol(name, VariableSymbol)
}

func (st *SymbolTable) DefineFunction(name string, startAddress int, paramNames []string) {
	symbol := Symbol{
		Name:       name,
		Type:       FunctionSymbol,
		ParamNames: paramNames,
	}
	st.Symbols[name] = symbol
}

func (st *SymbolTable) Resolve(name string) (Symbol, bool) {
	symbol, ok := st.Symbols[name]
	if !ok && st.Parent != nil {
		symbol, ok = st.Parent.Resolve(name)
	}
	return symbol, ok
}

func (st *SymbolTable) ResolveLocal(funcName, name string) (Symbol, bool) {
	if symbol, ok := st.Symbols[name]; ok {
		return symbol, ok
	}

	if st.Parent != nil {
		return st.Parent.ResolveLocal(funcName, name)
	}
	return Symbol{}, false
}

type Compiler struct {
	bytecode    []byte
	symbolTable *SymbolTable
}

func NewCompiler() *Compiler {
	return &Compiler{
		bytecode:    []byte{},
		symbolTable: NewSymbolTable(nil),
	}
}

func (c *Compiler) CompileASTByte(ast interface{}) ([]byte, error) {
	c.bytecode = []byte{}

	err := c.compileNode(ast)
	if err != nil {
		return nil, err
	}

	return c.bytecode, nil
}

func (c *Compiler) CompileAST(ast interface{}) ([]BytecodeInstruction, error) {
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
	//fmt.Println("Entering compileNode with node:", node)

	switch n := node.(type) {
	case []interface{}:
		if len(n) == 0 {
			return fmt.Errorf("empty expression")
		}

		if identifierNode, ok := n[0].(parser.Identifier); ok {
			switch identifierNode.Value {
			case "let":
				//fmt.Println("Handling 'def' statement")
				if len(n) != 3 {
					return fmt.Errorf("let expects two arguments")
				}
				varName, ok := n[1].(parser.Identifier)
				if !ok {
					return fmt.Errorf("expected a variable name as the second argument to let")
				}
				//fmt.Printf("Variable name for 'def': %s\n", varName.Value)

				err := c.compileNode(n[2])
				if err != nil {
					return err
				}
				//fmt.Printf("Emitting DEFINE_VARIABLE for %s\n", varName.Value)
				c.emit(DEFINE_VARIABLE, varName.Value)
				return nil
			case "def":
				// (def funFunction (param) ;do fun func stuff (ret optionalReturnValue))
				if len(n) < 3 {
					return fmt.Errorf("function definition syntax error")
				}
				funcName, ok := n[1].(parser.Identifier)
				if !ok {
					return fmt.Errorf("function name must be an identifier")
				}

				paramsNode, ok := n[2].([]interface{})
				if !ok {
					return fmt.Errorf("function parameters must be in a list")
				}

				var paramNames []string
				for _, param := range paramsNode {
					paramName, ok := param.(parser.Identifier)
					if !ok {
						return fmt.Errorf("invalid parameter name in function definition")
					}
					paramNames = append(paramNames, paramName.Value)
				}

				err := c.compileNode(n[3:])
				if err != nil {
					return err
				}

				c.emit(DEFINE_FUNCTION, funcName.Value, paramNames)

				return nil
			case "print":
				if len(n) != 2 {
					return fmt.Errorf("print expects one argument")
				}
				err := c.compileNode(n[1])
				if err != nil {
					return err
				}
				c.emit(PRINT)
				return nil

			}
		}

		if funcNameNode, ok := n[0].(parser.Identifier); ok {
			symbol, found := c.symbolTable.Resolve(funcNameNode.Value)
			if found && symbol.Type == FunctionSymbol {
				for _, arg := range n[1:] {
					if err := c.compileNode(arg); err != nil {
						return err
					}
				}
				c.emit(CALL_FUNCTION, funcNameNode.Value)
				return nil
			}
		}

		for _, operand := range n {
			if _, ok := n[0].(parser.Operator); ok {
				for _, operand := range n[1:] {
					err := c.compileNode(operand)
					if err != nil {
						return err
					}
				}
				return c.compileNode(n[0])
			}
			err := c.compileNode(operand)
			if err != nil {
				return err
			}
		}

	case parser.Identifier:
		c.emit(PUSH_VARIABLE, n.Value)
	case parser.Number:
		c.emit(PUSH_NUMBER, n.Value)
	case parser.Boolean:
		c.emit(PUSH_BOOL, n.Value)
	case parser.String:
		//fmt.Printf("Emitting String: %v\n", n.Value)
		strVal := strings.Trim(n.Value, "'")
		c.emit(PUSH_STRING, strVal)
	case parser.Operator:
		switch n.Value {
		case "+":
			c.emit(ADD)
		case "-":
			c.emit(SUB)
		case ">":
			c.emit(GRT)
		case "<":
			c.emit(LESS)
		case "=":
			c.emit(EQ)
		case "?":
			c.emit(NEQ)
		// ... other operators ...
		default:
			return fmt.Errorf("unknown operator: %s", n.Value)
		}
	case parser.IfStatement:
		ifStatement := n

		err := c.compileNode(ifStatement.Condition)
		if err != nil {
			return err
		}

		c.emit(IF)

		err = c.compileNode(ifStatement.ThenBlock)
		if err != nil {
			return err
		}

		if ifStatement.ElseBlock != nil {
			c.emit(ELSE)
			err = c.compileNode(ifStatement.ElseBlock)
			if err != nil {
				return err
			}
		}

		c.emit(ENDIF)

		return nil
	case parser.FunctionDefinition:
		return c.compileFunctionDefinition(n)
	default:
		return fmt.Errorf("unknown node type: %T", n)
	}

	//fmt.Println("Exiting compileNode")
	return nil
}

func (c *Compiler) compileFunctionDefinition(fnDef parser.FunctionDefinition) error {
	startAddress := len(c.bytecode)

	//jumpIndex := len(fnDef.Body) + 2 // also has to jump over JUMP and RETURN
	placeholderJumpIndex := len(c.bytecode)
	c.emit(JUMP, 0) // Placeholder offset

	c.enterScope()
	var paramNames []string
	for _, param := range fnDef.Params {
		c.symbolTable.DefineVariable(param)
		paramNames = append(paramNames, param)
	}

	instructionCountBeforeBody := len(c.bytecode)
	for _, expr := range fnDef.Body {
		if err := c.compileNode(expr); err != nil {
			return err
		}
	}
	instructionCountAfterBody := len(c.bytecode)

	if !c.endsInReturn(fnDef.Body) {
		c.emit(RETURN)
	}
	c.leaveScope()

	jumpOffset := instructionCountAfterBody - instructionCountBeforeBody
	updateJumpInstruction(c.bytecode, placeholderJumpIndex, jumpOffset)

	c.symbolTable.DefineFunction(fnDef.Name, startAddress, paramNames)
	paramCount := len(fnDef.Params)
	c.emitDefineFunction(fnDef.Name, startAddress, paramCount, paramNames)

	return nil
}

func updateJumpInstruction(bytecode []byte, jumpIndex int, offset int) {
	offsetBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(offsetBytes, uint32(offset))
	copy(bytecode[jumpIndex+1:], offsetBytes) // +1 to skip the opcode byte
}

func (c *Compiler) enterScope() {
	c.symbolTable = NewSymbolTable(c.symbolTable)
}

func (c *Compiler) leaveScope() {
	c.symbolTable = c.symbolTable.Parent
}

func (c *Compiler) endsInReturn(body []interface{}) bool {
	if len(body) == 0 {
		return false
	}
	lastExpr := body[len(body)-1]
	_, isRet := lastExpr.(parser.ReturnStatement)
	return isRet
}

func (c *Compiler) emitDefineFunction(funcName string, startAddress, paramCount int, paramNames []string) {
	c.emit(DEFINE_FUNCTION, funcName, startAddress, paramCount, paramNames)
}

func (c *Compiler) emit(opcode Opcode, operands ...interface{}) {
	//fmt.Printf("Emitting opcode: %d with operands: %v\n", opcode, operands)
	opcodeBytes := []byte{byte(opcode)}
	operandBytes := serializeOperands(operands)
	c.bytecode = append(c.bytecode, opcodeBytes...)
	c.bytecode = append(c.bytecode, operandBytes...)
}

func serializeOperands(operands []interface{}) []byte {
	var result []byte

	for _, operand := range operands {
		//fmt.Printf("Serializing Operand: %v, Type: %T\n", operand, operand)

		switch v := operand.(type) {
		case int:
			intBytes := make([]byte, 4)
			binary.LittleEndian.PutUint32(intBytes, uint32(v))
			result = append(result, intBytes...)
		case float64:
			//fmt.Printf("Serializing floating-point number:%v\n", operand)
			bits := math.Float64bits(v)
			buf := make([]byte, 8)
			binary.LittleEndian.PutUint64(buf, bits)
			result = append(result, buf...)
			//fmt.Printf("converted bytes:%v\n", result)
		case string:
			//fmt.Printf("Serializing string operand: %s\n", v)
			strBytes := []byte(v)
			lengthBuf := make([]byte, 4)
			binary.LittleEndian.PutUint32(lengthBuf, uint32(len(strBytes)))
			//fmt.Println(lengthBuf)
			//fmt.Println(strBytes)
			result = append(result, lengthBuf...)
			result = append(result, strBytes...)
		case bool:
			if v {
				result = append(result, 1)
			} else {
				result = append(result, 0)
			}
		case []string:
			sliceLenBytes := make([]byte, 4)
			binary.LittleEndian.PutUint32(sliceLenBytes, uint32(len(v)))
			result = append(result, sliceLenBytes...)

			//fmt.Printf("===serializing params: %v\n", v)
			for _, str := range v {
				strBytes := []byte(str)
				lengthBuf := make([]byte, 4)
				binary.LittleEndian.PutUint32(lengthBuf, uint32(len(strBytes)))
				result = append(result, lengthBuf...)
				result = append(result, strBytes...)
				//fmt.Printf("===serialized param: %v\tlength: %v\tbytes: %v\n", str, lengthBuf, strBytes)
			}
		default:
			fmt.Printf("Unsupported operand type: %T\n", v)
		}
	}

	return result
}

func calculateCorrectedOffset(bytecode []byte, offset int) int {
	count := 0
	for i := 0; i < offset && i < len(bytecode); {
		opcode := Opcode(bytecode[i])
		i++
		switch opcode {
		case PUSH_NUMBER:
			i += 8 
		case PUSH_BOOL, PUSH_VARIABLE, DEFINE_VARIABLE:
			if i+4 > len(bytecode) {
				break
			}
			operandLength := int(binary.LittleEndian.Uint32(bytecode[i : i+4]))
			i += 4 + operandLength
			// ... other cases ...
		}
		count++ 
	}
	count++
	return count
}

func convertBytecode(rawBytecode []byte) ([]BytecodeInstruction, error) {
	fmt.Printf("Raw Bytecode: %v\n", rawBytecode)
	var instructions []BytecodeInstruction
	i := 0

	for i < len(rawBytecode) {
		opcode := Opcode(rawBytecode[i])
		i++

		var operands []interface{}
		switch opcode {
		case PUSH_NUMBER:
			if i+8 > len(rawBytecode) {
				return nil, fmt.Errorf("invalid bytecode, unexpected end of data")
			}
			numberBytes := rawBytecode[i : i+8]
			operands = append(operands, numberBytes)
			i += 8
		case PUSH_BOOL:
			if i >= len(rawBytecode) {
				return nil, fmt.Errorf("invalid bytecode, unexpected end of data")
			}
			boolByte := rawBytecode[i]
			i++
			operands = append(operands, boolByte)

		case PUSH_STRING:
			if i+4 > len(rawBytecode) {
				return nil, fmt.Errorf("invalid bytecode, unexpected end of data")
			}
			strLen := int(binary.LittleEndian.Uint32(rawBytecode[i : i+4]))
			i += 4

			if i+strLen > len(rawBytecode) {
				return nil, fmt.Errorf("invalid bytecode, unexpected end of data")
			}
			strBytes := rawBytecode[i : i+strLen]
			operands = append(operands, strBytes)
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
			varBytes := rawBytecode[i : i+varNameLen]
			operands = append(operands, varBytes)
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
			varBytes := rawBytecode[i : i+varNameLen]
			operands = append(operands, varBytes)
			i += varNameLen
		case DEFINE_FUNCTION:
			if i+4 > len(rawBytecode) {
				return nil, fmt.Errorf("invalid bytecode, unexpected end of data for function name length")
			}
			funcNameLen := int(binary.LittleEndian.Uint32(rawBytecode[i : i+4]))
			i += 4

			if i+funcNameLen > len(rawBytecode) {
				return nil, fmt.Errorf("invalid bytecode, unexpected end of data for function name")
			}
			funcNameBytes := rawBytecode[i : i+funcNameLen]
			i += funcNameLen

			if i+4 > len(rawBytecode) {
				return nil, fmt.Errorf("invalid bytecode, unexpected end of data for start address")
			}
			startAddressBytes := rawBytecode[i : i+4]
			i += 4

			if i+4 > len(rawBytecode) {
				return nil, fmt.Errorf("invalid bytecode, unexpected end of data for parameter count")
			}
			paramCountBytes := rawBytecode[i : i+4]
			i += 4
			operands = append(operands, funcNameBytes, startAddressBytes, paramCountBytes)
			//fmt.Printf("===appepnded funcName, startAddress, paramCount: %v\n", operands)

			paramCount := binary.LittleEndian.Uint32(paramCountBytes)

			if i+4 > len(rawBytecode) {
				return nil, fmt.Errorf("invalid bytecode, unexpected end of data for parameter list length")
			}
			paramNamesLenBytes := rawBytecode[i : i+4]
			paramNamesLen := binary.LittleEndian.Uint32(paramNamesLenBytes)
			if paramNamesLen != paramCount {
				return nil, fmt.Errorf("invalid bytecode, parameter list must have the same length as parameter count")
			}
			operands = append(operands, paramNamesLenBytes)
			//fmt.Printf("===appended param list len: %v\n", operands)
			i += 4
			//var paramNamesBytes []interface{}
			for j := uint32(0); j < paramCount; j++ {

				if i+4 > len(rawBytecode) {
					return nil, fmt.Errorf("invalid bytecode, unexpected end of data for parameter name length")
				}
				paramNameLenBytes := rawBytecode[i : i+4]
				paramNameLen := int(binary.LittleEndian.Uint32(paramNameLenBytes))
				if i+paramNameLen > len(rawBytecode) {
					return nil, fmt.Errorf("invalid bytecode, unexpected end of data for parameter name")
				}
				operands = append(operands, paramNameLenBytes)
				//fmt.Printf("===appended param name len: %v\n", operands)
				i += 4
				paramNameBytes := rawBytecode[i : i+paramNameLen]
				i += paramNameLen

				operands = append(operands, paramNameBytes)
				//fmt.Printf("===appended param name bytes: %v\n", paramNameBytes)
				//fmt.Printf("===after appending param name bytes: %v\n", operands)
			}
		case CALL_FUNCTION:
			if i+4 > len(rawBytecode) {
				return nil, fmt.Errorf("invalid bytecode, unexpected end of data")
			}
			funcNameLen := int(binary.LittleEndian.Uint32(rawBytecode[i : i+4]))
			i += 4

			if i+funcNameLen > len(rawBytecode) {
				return nil, fmt.Errorf("invalid bytecode, unexpected end of data")
			}
			funcNameBytes := rawBytecode[i : i+funcNameLen]
			operands = append(operands, funcNameBytes)
			i += funcNameLen
		case JUMP:
			if i+4 > len(rawBytecode) {
				return nil, fmt.Errorf("invalid bytecode, unexpected end of data for JUMP offset")
			}
			jumpOffsetBytes := rawBytecode[i : i+4]
			i += 4
			jumpOffset := int(binary.LittleEndian.Uint32(jumpOffsetBytes))
			correctedOffset := calculateCorrectedOffset(rawBytecode[i:], jumpOffset)
			offsetBytes := make([]byte, 4)
			binary.LittleEndian.PutUint32(offsetBytes, uint32(correctedOffset))
			operands = append(operands, offsetBytes)

		default:
			// Opcodes without operands
		}

		instructions = append(instructions, BytecodeInstruction{Opcode: opcode, Operands: operands})
	}

	return instructions, nil
}
