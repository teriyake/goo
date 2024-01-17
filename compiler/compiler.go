package compiler

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"teriyake/goo/parser"
)

type Opcode int

const (
	ADD Opcode = iota
	SUB
	MUL
	DIV
	GRT
	LESS
	EQ
	NEQ
	PUSH_VARIABLE Opcode = iota + 20
	PUSH_NUMBER
	PUSH_BOOL
	PUSH_STRING
	DEFINE_VARIABLE
	DEFINE_FUNCTION
	CREATE_LAMBDA
	CALL_LAMBDA
	IF Opcode = iota + 30
	ELSE
	ENDIF
	PRINT
	RETURN
	JUMP
	CALL_FUNCTION
	MAP = iota + 40
)

func OpcodeToString(op Opcode) string {
	opcodeStrings := map[Opcode]string{
		ADD:             "ADD",
		SUB:             "SUB",
		MUL:             "MUL",
		DIV:             "DIV",
		GRT:             "GRT",
		LESS:            "LESS",
		EQ:              "EQ",
		NEQ:             "NEQ",
		PUSH_VARIABLE:   "PUSH_VARIABLE",
		PUSH_NUMBER:     "PUSH_NUMBER",
		PUSH_BOOL:       "PUSH_BOOL",
		PUSH_STRING:     "PUSH_STRING",
		DEFINE_VARIABLE: "DEFINE_VARIABLE",
		DEFINE_FUNCTION: "DEFINE_FUNCTION",
		CREATE_LAMBDA:   "CREATE_LAMBDA",
		CALL_LAMBDA:     "CALL_LAMBDA",
		IF:              "IF",
		ELSE:            "ELSE",
		ENDIF:           "ENDIF",
		PRINT:           "PRINT",
		RETURN:          "RETURN",
		JUMP:            "JUMP",
		CALL_FUNCTION:   "CALL_FUNCTION",
		MAP:             "MAP",
	}

	return opcodeStrings[op]
}

type BytecodeInstruction struct {
	Opcode   Opcode
	Operands []interface{}
}

type DataType int

const (
	IntType DataType = iota
	FloatType
	StringType
	BoolType
)

func ParseDataType(pt string) DataType {
	switch pt {
	case "int":
		return IntType
	case "float":
		return FloatType
	case "string":
		return StringType
	case "bool":
		return BoolType
	// more cases later
	default:
		return IntType // default to IntType for unknown types??
	}
}

type SymbolType int

const (
	VariableSymbol SymbolType = iota
	FunctionSymbol
)

type Symbol struct {
	Name         string
	Type         SymbolType
	DataType     DataType
	ParamNames   []string
	StartAddress int
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

func (st *SymbolTable) Print() {
	fmt.Println("Symbol Table:")
	for name, symbol := range st.Symbols {
		fmt.Printf("Name: %s, Type: %s", name, symbol.Type)
		if symbol.Type == FunctionSymbol {
			fmt.Printf(", Param Names: %v", symbol.ParamNames)
		}
		fmt.Printf(", Start Address: %d\n", symbol.StartAddress)
	}
}

func (st *SymbolTable) DefineSymbol(name string, symbolType SymbolType, dataType DataType) {
	st.Symbols[name] = Symbol{
		Name:     name,
		Type:     symbolType,
		DataType: dataType,
	}
}

func (st *SymbolTable) DefineVariable(name string, dataType DataType) {
	st.DefineSymbol(name, VariableSymbol, dataType)
}

func (st *SymbolTable) DefineFunction(name string, startAddress int, paramNames []string, returnType DataType) {
	symbol := Symbol{
		Name:         name,
		Type:         FunctionSymbol,
		ParamNames:   paramNames,
		StartAddress: startAddress,
		DataType:     returnType,
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

func (st *SymbolTable) IsLocal(name string) bool {
	_, found := st.Symbols[name]
	return found
}

func (st *SymbolTable) IsFunction(name string) bool {
	symbol, ok := st.Resolve(name)
	if !ok {
		return false
	}
	return symbol.Type == FunctionSymbol
}

func (st *SymbolTable) IsFunctionParameter(funcName, name string) bool {
	symbol, ok := st.ResolveLocal(funcName, name)
	if !ok {
		return false
	}
	return symbol.Type == VariableSymbol
}

func (st *SymbolTable) UpdateFunctionStartAddress(name string, address int) {
	if symbol, ok := st.Symbols[name]; ok && symbol.Type == FunctionSymbol {
		symbol.StartAddress = address
		st.Symbols[name] = symbol
	}
}

type Compiler struct {
	bytecode        []byte
	symbolTable     *SymbolTable
	currentFunction string
	insideFunction  bool
}

func NewCompiler() *Compiler {
	return &Compiler{
		bytecode:        []byte{},
		symbolTable:     NewSymbolTable(nil),
		currentFunction: "",
		insideFunction:  false,
	}
}

func (c *Compiler) setCurrentFunction(functionName string) {
	c.currentFunction = functionName
}

func (c *Compiler) getCurrentFunction() string {
	return c.currentFunction
}

func (c *Compiler) isInsideFunction() bool {
	return c.currentFunction != ""
}

func (c *Compiler) CompileASTByte(ast interface{}) ([]byte, error) {
	c.bytecode = []byte{}

	err := c.compileNode(ast)
	if err != nil {
		return nil, err
	}

	return c.bytecode, nil
}

func (c *Compiler) CompileAST(ast interface{}) ([]BytecodeInstruction, map[int]int, error) {
	c.bytecode = []byte{}

	err := c.compileNode(ast)
	if err != nil {
		return nil, nil, err
	}

	bytecodeInstructions, offsetMap, err := convertBytecode(c.bytecode)
	if err != nil {
		return nil, nil, err
	}

	return bytecodeInstructions, offsetMap, nil
}

func (c *Compiler) compileNode(node interface{}) error {
	//fmt.Println("Entering compileNode with node:", node)

	switch n := node.(type) {
	case []interface{}:
		if len(n) == 0 {
			return fmt.Errorf("empty expression")
		}

		if varNode, ok := n[0].(parser.TypeAnnotation); ok {

			if len(n) != 2 {
				return fmt.Errorf("let expects two arguments")
			}
			varName := varNode.Variable
			varType := varNode.Type

			err := c.compileNode(n[1])
			if err != nil {
				return err
			}

			c.symbolTable.DefineVariable(varName, ParseDataType(varType))

			c.symbolTable.Print()
			//fmt.Printf("Emitting DEFINE_VARIABLE for %s\n", varName.Value)
			c.emit(DEFINE_VARIABLE, varName)
			return nil
		}

		if identifierNode, ok := n[0].(parser.Identifier); ok {
			switch identifierNode.Value {
			case "def":
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
		symbol, found := c.symbolTable.Resolve(n.Value)
		if found {
			if symbol.Type == FunctionSymbol {
				c.emit(CALL_FUNCTION, n.Value)
			} else if symbol.Type == VariableSymbol {
				if c.isInsideFunction() && c.symbolTable.IsFunctionParameter(c.currentFunction, n.Value) {
					c.emit(PUSH_VARIABLE, n.Value)
				} else {
					c.emit(PUSH_VARIABLE, n.Value)
				}
			}
		} else {
			return fmt.Errorf("undefined identifier: %s", n.Value)
		}
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
		case "*":
			c.emit(MUL)
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
	case parser.ReturnStatement:
		err := c.compileNode(n.ReturnValue)
		if err != nil {
			return err
		}
		c.emit(RETURN)
		return nil
	case parser.LambdaExpression:
		lambdaExpr := node.(parser.LambdaExpression)
		paramNames := make([]string, len(lambdaExpr.Params))

		jumpInstructionIndex := len(c.bytecode)
		c.emit(JUMP, 0)
		c.enterScope()

		for i, param := range lambdaExpr.Params {
			paramNames[i] = param.Variable
			c.symbolTable.DefineVariable(param.Variable, ParseDataType(param.Type))
		}

		startAddress := len(c.bytecode)
		for _, expr := range lambdaExpr.Body {
			err := c.compileNode(expr)
			if err != nil {
				return err
			}
		}
		endAddress := len(c.bytecode)
		jumpOffset := len(c.bytecode) - jumpInstructionIndex
		correctOffset := calculateCorrectedOffset(c.bytecode, jumpOffset) - 1
		updateJumpInstruction(c.bytecode, jumpInstructionIndex, correctOffset)

		//fmt.Printf("Symbol Table before capturing lambda variables: \n")
		c.symbolTable.Print()
		capturedVariables, err := c.determineCapturedVariables(lambdaExpr.Body, lambdaExpr.Params)
		if err != nil {
			return fmt.Errorf("Error capturing lambda variables: %v\n", err)
		}
		//fmt.Printf("Captured lambda variables: %v\n", capturedVariables)

		c.emit(CREATE_LAMBDA, startAddress, endAddress, len(lambdaExpr.Params), paramNames, len(capturedVariables), capturedVariables)

		c.leaveScope()

		return nil
	case parser.LambdaCall:
		lambdaCall := node.(parser.LambdaCall)

		err := c.compileNode(lambdaCall.Lambda)
		if err != nil {
			return err
		}

		for _, arg := range lambdaCall.Arguments {
			err := c.compileNode(arg)
			if err != nil {
				return err
			}
		}

		c.emit(CALL_LAMBDA, len(lambdaCall.Arguments))

		return nil
	case parser.MapExpression:
		return c.compileMapExpression(n)

	default:
		return fmt.Errorf("unknown node type: %T", n)
	}

	//fmt.Println("Exiting compileNode")
	return nil
}

func (c *Compiler) compileMapExpression(mapExpr parser.MapExpression) error {
	err := c.compileNode(mapExpr.Lambda)
	if err != nil {
		return fmt.Errorf("error compiling lambda in map expression: %v", err)
	}

	for _, arg := range mapExpr.Arguments {
		err := c.compileNode(arg)
		if err != nil {
			return fmt.Errorf("error compiling argument in map expression: %v", err)
		}
	}

	c.emit(MAP, len(mapExpr.Arguments))

	return nil
}

func (c *Compiler) compileFunctionDefinition(fnDef parser.FunctionDefinition) error {
	fmt.Println("Compiling function definition:", fnDef.Name)
	startAddress := len(c.bytecode)

	var paramNames []string
	for _, param := range fnDef.Params {
		paramNames = append(paramNames, param.Variable)
		c.symbolTable.DefineVariable(param.Variable, ParseDataType(param.Type))
		fmt.Printf("Defined variable: %s\n", param)
	}
	c.symbolTable.DefineFunction(fnDef.Name, startAddress, paramNames, ParseDataType(fnDef.ReturnType))

	fmt.Println("Symbol table after defining parameters:")
	c.symbolTable.Print()

	c.emit(JUMP, 0)

	c.enterScope()
	c.setCurrentFunction(fnDef.Name)

	instructionCountBeforeBody := len(c.bytecode)
	for _, expr := range fnDef.Body {
		switch e := expr.(type) {
		case parser.ReturnStatement:
			if err := c.compileNode(e.ReturnValue); err != nil {
				return err
			}
			c.emit(RETURN)
		default:
			if err := c.compileNode(expr); err != nil {
				return err
			}
		}
	}
	if !c.endsInReturn(fnDef.Body) {
		c.emit(RETURN)
	}
	instructionCountAfterBody := len(c.bytecode)
	jumpOffset := instructionCountAfterBody - instructionCountBeforeBody
	correctOffset := calculateCorrectedOffset(c.bytecode, jumpOffset) - 1
	updateJumpInstruction(c.bytecode, startAddress, correctOffset)
	c.leaveScope()

	fmt.Println("Symbol table after leaving scope:")
	c.symbolTable.Print()

	c.setCurrentFunction("")
	c.symbolTable.DefineFunction(fnDef.Name, startAddress, paramNames, ParseDataType(fnDef.ReturnType))
	paramCount := len(fnDef.Params)
	c.emitDefineFunction(fnDef.Name, startAddress, paramCount, paramNames)
	fmt.Println("Function compiled:", fnDef.Name)
	return nil
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
		switch v := operand.(type) {
		case int:
			intBytes := make([]byte, 4)
			binary.LittleEndian.PutUint32(intBytes, uint32(v))
			result = append(result, intBytes...)
		case float64:
			bits := math.Float64bits(v)
			buf := make([]byte, 8)
			binary.LittleEndian.PutUint64(buf, bits)
			result = append(result, buf...)
		case string:
			strBytes := []byte(v)
			lengthBuf := make([]byte, 4)
			binary.LittleEndian.PutUint32(lengthBuf, uint32(len(strBytes)))
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

			for _, str := range v {
				strBytes := []byte(str)
				lengthBuf := make([]byte, 4)
				binary.LittleEndian.PutUint32(lengthBuf, uint32(len(strBytes)))
				result = append(result, lengthBuf...)
				result = append(result, strBytes...)
			}
		default:
			fmt.Printf("Unsupported operand type: %T\n", v)
		}
	}

	return result
}

func calculateCorrectedOffset(bytecode []byte, targetOffset int) int {
	instructionCount := 0
	i := 0

	for i < len(bytecode) && instructionCount < targetOffset {
		opcode := Opcode(bytecode[i])
		i++

		switch opcode {
		case PUSH_NUMBER:
			i += 8
		case PUSH_STRING, PUSH_VARIABLE, DEFINE_VARIABLE, CALL_FUNCTION:
			if i+4 > len(bytecode) {
				break
			}
			operandLength := int(binary.LittleEndian.Uint32(bytecode[i : i+4]))
			i += 4 + operandLength
		case PUSH_BOOL:
			i++
		case JUMP:
			i += 4

		default:

		}

		instructionCount++
	}

	return instructionCount
}

func updateJumpInstruction(bytecode []byte, jumpIndex int, correctedOffset int) {
	offsetBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(offsetBytes, uint32(correctedOffset))
	copy(bytecode[jumpIndex+1:], offsetBytes)
}

func (c *Compiler) determineCapturedVariables(lambdaBody []interface{}, lambdaParams []parser.TypeAnnotation) ([]string, error) {
	captured := make(map[string]bool)
	var err error

	paramNames := make(map[string]bool)
	for _, param := range lambdaParams {
		paramNames[param.Variable] = true
	}

	var visitNode func(node interface{})
	visitNode = func(node interface{}) {
		switch n := node.(type) {
		case parser.Identifier:
			symbol, found := c.symbolTable.Resolve(n.Value)
			if found && symbol.Type == VariableSymbol && !paramNames[n.Value] {
				captured[n.Value] = true
			}
		case []interface{}:
			for _, elem := range n {
				visitNode(elem)
			}
		}
	}

	for _, expr := range lambdaBody {
		visitNode(expr)
	}

	var capturedVars []string
	for varName := range captured {
		capturedVars = append(capturedVars, varName)
	}

	return capturedVars, err
}

func convertBytecode(rawBytecode []byte) ([]BytecodeInstruction, map[int]int, error) {
	fmt.Printf("Raw Bytecode: %v\n", rawBytecode)
	var instructions []BytecodeInstruction
	offsetToInstructionIndex := make(map[int]int)
	i := 0
	currentOffset := 0

	for i < len(rawBytecode) {
		offsetToInstructionIndex[currentOffset] = len(instructions)
		opcode := Opcode(rawBytecode[i])
		i++
		currentOffset++

		var operands []interface{}
		switch opcode {
		case PUSH_NUMBER:
			if i+8 > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data")
			}
			numberBytes := rawBytecode[i : i+8]
			operands = append(operands, numberBytes)
			i += 8
			currentOffset += 8
		case PUSH_BOOL:
			if i >= len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data")
			}
			boolByte := rawBytecode[i]
			operands = append(operands, boolByte)
			i++
			currentOffset++
		case PUSH_STRING:
			if i+4 > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data")
			}
			strLenBytes := rawBytecode[i : i+4]
			operands = append(operands, strLenBytes)
			i += 4
			currentOffset += 4

			strLen := int(binary.LittleEndian.Uint32(strLenBytes))

			if i+strLen > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data")
			}
			strBytes := rawBytecode[i : i+strLen]
			operands = append(operands, strBytes)
			i += strLen
			currentOffset += strLen
		case PUSH_VARIABLE:
			if i+4 > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data")
			}
			varNameLenBytes := rawBytecode[i : i+4]
			operands = append(operands, varNameLenBytes)
			i += 4
			currentOffset += 4

			varNameLen := int(binary.LittleEndian.Uint32(varNameLenBytes))
			if i+varNameLen > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data")
			}
			varBytes := rawBytecode[i : i+varNameLen]
			operands = append(operands, varBytes)
			i += varNameLen
			currentOffset += varNameLen
		case DEFINE_VARIABLE:
			if i+4 > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data")
			}
			varNameLenBytes := rawBytecode[i : i+4]
			operands = append(operands, varNameLenBytes)
			i += 4
			currentOffset += 4

			varNameLen := int(binary.LittleEndian.Uint32(varNameLenBytes))
			if i+varNameLen > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data")
			}
			varBytes := rawBytecode[i : i+varNameLen]
			operands = append(operands, varBytes)
			i += varNameLen
			currentOffset += varNameLen
		case DEFINE_FUNCTION:
			if i+4 > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data for function name length")
			}
			funcNameLen := int(binary.LittleEndian.Uint32(rawBytecode[i : i+4]))
			i += 4
			currentOffset += 4

			if i+funcNameLen > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data for function name")
			}
			funcNameBytes := rawBytecode[i : i+funcNameLen]
			i += funcNameLen
			currentOffset += funcNameLen

			if i+4 > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data for start address")
			}
			startAddressBytes := rawBytecode[i : i+4]
			i += 4
			currentOffset += 4

			if i+4 > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data for parameter count")
			}
			paramCountBytes := rawBytecode[i : i+4]
			i += 4
			currentOffset += 4
			operands = append(operands, funcNameBytes, startAddressBytes, paramCountBytes)
			//fmt.Printf("===appepnded funcName, startAddress, paramCount: %v\n", operands)

			paramCount := binary.LittleEndian.Uint32(paramCountBytes)

			if i+4 > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data for parameter list length")
			}
			paramNamesLenBytes := rawBytecode[i : i+4]
			paramNamesLen := binary.LittleEndian.Uint32(paramNamesLenBytes)
			if paramNamesLen != paramCount {
				return nil, nil, fmt.Errorf("invalid bytecode, parameter list must have the same length as parameter count")
			}
			operands = append(operands, paramNamesLenBytes)
			//fmt.Printf("===appended param list len: %v\n", operands)
			i += 4
			currentOffset += 4
			//var paramNamesBytes []interface{}
			for j := uint32(0); j < paramCount; j++ {

				if i+4 > len(rawBytecode) {
					return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data for parameter name length")
				}
				paramNameLenBytes := rawBytecode[i : i+4]
				paramNameLen := int(binary.LittleEndian.Uint32(paramNameLenBytes))
				if i+paramNameLen > len(rawBytecode) {
					return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data for parameter name")
				}
				operands = append(operands, paramNameLenBytes)
				//fmt.Printf("===appended param name len: %v\n", operands)
				i += 4
				currentOffset += 4
				paramNameBytes := rawBytecode[i : i+paramNameLen]
				i += paramNameLen
				currentOffset += paramNameLen

				operands = append(operands, paramNameBytes)
				//fmt.Printf("===appended param name bytes: %v\n", paramNameBytes)
				//fmt.Printf("===after appending param name bytes: %v\n", operands)
			}
		case CALL_FUNCTION:
			if i+4 > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data")
			}
			funcNameLenBytes := rawBytecode[i : i+4]
			operands = append(operands, funcNameLenBytes)
			i += 4
			currentOffset += 4

			funcNameLen := int(binary.LittleEndian.Uint32(funcNameLenBytes))
			if i+funcNameLen > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data")
			}
			funcNameBytes := rawBytecode[i : i+funcNameLen]
			operands = append(operands, funcNameBytes)
			i += funcNameLen
			currentOffset += funcNameLen
		case JUMP:
			if i+4 > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data for JUMP offset")
			}
			jumpOffsetBytes := rawBytecode[i : i+4]
			operands = append(operands, jumpOffsetBytes)
			i += 4
			currentOffset += 4
		case CREATE_LAMBDA:
			if i+4 > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data for lambda start address")
			}
			startAddressBytes := rawBytecode[i : i+4]
			operands = append(operands, startAddressBytes)
			i += 4
			currentOffset += 4

			if i+4 > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data for lambda end address")
			}
			endAddressBytes := rawBytecode[i : i+4]
			operands = append(operands, endAddressBytes)
			i += 4
			currentOffset += 4

			if i+4 > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data for number of lambda params")
			}
			numParamBytes := rawBytecode[i : i+4]
			operands = append(operands, numParamBytes)
			i += 4
			currentOffset += 4

			if i+4 > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data for slice length of lambda params")
			}
			paramSliceLenBytes := rawBytecode[i : i+4]
			i += 4
			currentOffset += 4

			numParam := int(binary.LittleEndian.Uint32(numParamBytes))
			paramSliceLen := int(binary.LittleEndian.Uint32(paramSliceLenBytes))
			if paramSliceLen != numParam {
				return nil, nil, fmt.Errorf("invalid bytecode, invalid data for slice length of lambda params")
			}

			for jj := 0; jj < numParam; jj++ {
				if i+4 > len(rawBytecode) {
					return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data for lambda param name length")
				}
				paramNameLenBytes := rawBytecode[i : i+4]
				paramNameLen := int(binary.LittleEndian.Uint32(paramNameLenBytes))
				operands = append(operands, paramNameLenBytes)
				i += 4
				currentOffset += 4

				if i+paramNameLen > len(rawBytecode) {
					return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data for lambda param name")
				}
				paramNameBytes := rawBytecode[i : i+paramNameLen]
				operands = append(operands, paramNameBytes)
				i += paramNameLen
				currentOffset += paramNameLen
			}

			if i+4 > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data for number of captured variables")
			}
			numCapturedBytes := rawBytecode[i : i+4]
			numCaptured := int(binary.LittleEndian.Uint32(numCapturedBytes))
			operands = append(operands, numCapturedBytes)
			i += 4
			currentOffset += 4

			if i+4 > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data for slice length of captured variables")
			}
			/*
				numCapturedSliceLenBytes := rawBytecode[i : i+4]
				numCapturedSliceLen := int(binary.LittleEndian.Uint32(numCapturedSliceLenBytes))
				operands = append(operands, numCapturedBytes)
			*/
			i += 4
			currentOffset += 4

			for j := 0; j < numCaptured; j++ {
				if i+4 > len(rawBytecode) {
					return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data for captured variable name length")
				}
				varNameLenBytes := rawBytecode[i : i+4]
				varNameLen := int(binary.LittleEndian.Uint32(varNameLenBytes))
				operands = append(operands, varNameLenBytes)
				i += 4
				currentOffset += 4

				if i+varNameLen > len(rawBytecode) {
					return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data for captured variable name")
				}
				varNameBytes := rawBytecode[i : i+varNameLen]
				operands = append(operands, varNameBytes)
				i += varNameLen
				currentOffset += varNameLen
			}
		case CALL_LAMBDA:
			if i+4 > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data")
			}
			argLenBytes := rawBytecode[i : i+4]
			operands = append(operands, argLenBytes)
			i += 4
			currentOffset += 4
		case MAP:
			if i+4 > len(rawBytecode) {
				return nil, nil, fmt.Errorf("invalid bytecode, unexpected end of data")
			}
			argLenBytes := rawBytecode[i : i+4]
			operands = append(operands, argLenBytes)
			i += 4
			currentOffset += 4

		default:
			// Opcodes without operands
		}

		instructions = append(instructions, BytecodeInstruction{Opcode: opcode, Operands: operands})
	}

	return instructions, offsetToInstructionIndex, nil
}
