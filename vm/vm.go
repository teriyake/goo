package vm

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
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

func (rst *RuntimeSymbolTable) Print(indent string) {
	if rst == nil {
		fmt.Println(indent + "Symbol Table: nil")
		return
	}

	fmt.Println(indent + "Symbol Table:")
	for name, value := range rst.symbols {
		fmt.Printf("%s  %s: %v\n", indent, name, value)
	}

	if rst.parent != nil {
		fmt.Println(indent + "Parent:")
		rst.parent.Print(indent + "  ")
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

func (fm FunctionMetadata) Print() {
	fmt.Printf("FunctionMetadata - Start Address: %d, Param Count: %d, Param Names: %v\n", fm.StartAddress, fm.ParamCount, fm.ParamNames)
}

type LambdaFunction struct {
	StartAddress int
	EndAddress   int
	ParamCount   int
	ParamNames   []string
	CapturedVars []string
	SymbolTable  *RuntimeSymbolTable
}

func NewLambdaFunction(startAddress, endAddress, paramCount int, paramNames []string, capturedVars []string, symbolTable *RuntimeSymbolTable) *LambdaFunction {
	return &LambdaFunction{
		StartAddress: startAddress,
		EndAddress:   endAddress,
		ParamCount:   paramCount,
		ParamNames:   paramNames,
		CapturedVars: capturedVars,
		SymbolTable:  symbolTable,
	}
}

func (lf *LambdaFunction) Print(indent string) {
	fmt.Printf("%sLambdaFunction:\n", indent)
	fmt.Printf("%s  Start Address: %d\n", indent, lf.StartAddress)
	fmt.Printf("%s  End Address: %d\n", indent, lf.EndAddress)
	fmt.Printf("%s  Param Count: %d\n", indent, lf.ParamCount)
	fmt.Printf("%s  Param Names: %d\n", indent, lf.ParamNames)
	fmt.Printf("%s  Captured Vars: %v\n", indent, lf.CapturedVars)
	fmt.Printf("%s  SymbolTable:\n", indent)
	lf.SymbolTable.Print(indent + "    ")
}

func (lf *LambdaFunction) Call(vm *VM, args []interface{}) (interface{}, error) {
	if len(args) != lf.ParamCount {
		return nil, fmt.Errorf("lambda function expects %d arguments, got %d", lf.ParamCount, len(args))
	}

	lambdaSymbolTable := NewRuntimeSymbolTable(lf.SymbolTable)
	for i, paramName := range lf.CapturedVars {
		lambdaSymbolTable.Set(paramName, args[i])
	}

	savedPC := vm.pc
	//savedSymbolTable := vm.symbolTableStack[len(vm.symbolTableStack)-1]

	vm.symbolTableStack = append(vm.symbolTableStack, lambdaSymbolTable)
	err := vm.Run(lf.StartAddress)
	if err != nil {
		return nil, err
	}

	vm.pc = savedPC
	vm.symbolTableStack = vm.symbolTableStack[:len(vm.symbolTableStack)-1]

	if len(vm.stack) == 0 {
		return nil, fmt.Errorf("lambda function did not return a value")
	}
	result := vm.stack[len(vm.stack)-1]
	vm.stack = vm.stack[:len(vm.stack)-1]

	return result, nil
}

type CallStackEntry struct {
	returnAddress int
	symbolTable   *RuntimeSymbolTable
}

func (cse CallStackEntry) Print() {
	fmt.Printf("CallStackEntry - Return Address: %d\n", cse.returnAddress)
	fmt.Println("Symbol Table at this level:")
	cse.symbolTable.Print("  ")
}

type VM struct {
	stack            []interface{}
	pc               int
	code             []compiler.BytecodeInstruction
	offsetMap        map[int]int
	symbolTableStack []*RuntimeSymbolTable
	functions        map[string]FunctionMetadata
	callStack        []CallStackEntry
}

func NewVM(code []compiler.BytecodeInstruction, offsetMap map[int]int) *VM {
	globalSymbolTable := NewRuntimeSymbolTable(nil)
	return &VM{
		stack:            make([]interface{}, 0),
		pc:               0,
		code:             code,
		offsetMap:        offsetMap,
		symbolTableStack: []*RuntimeSymbolTable{globalSymbolTable},
		functions:        make(map[string]FunctionMetadata),
		callStack:        make([]CallStackEntry, 0),
	}
}

func (vm *VM) Print() {
	fmt.Println("VM State:")
	fmt.Printf("  Program Counter: %d\n", vm.pc)
	fmt.Printf("  Stack: %v\n", vm.stack)

	fmt.Println("  Symbol Table Stack:")
	for _, st := range vm.symbolTableStack {
		st.Print("    ")
	}

	fmt.Println("  Function Metadata:")
	for name, fm := range vm.functions {
		fmt.Printf("    %s: ", name)
		fm.Print()
	}

	fmt.Println("  Call Stack:")
	if len(vm.callStack) == 0 {
		fmt.Println("    EMPTY")
	} else {
		for _, cse := range vm.callStack {
			cse.Print()
		}
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

func (vm *VM) Run(optionalStartEndAddress ...int) error {
	start := 0
	end := len(vm.code)
	if len(optionalStartEndAddress) == 2 {
		start = optionalStartEndAddress[0]
		end = optionalStartEndAddress[1]
	} else if len(optionalStartEndAddress) == 1 {
		start = optionalStartEndAddress[0]
	}

	for vm.pc = start; vm.pc < end; vm.pc++ {
		instruction := vm.code[vm.pc]
		fmt.Printf("Executing Instruction at PC %v: Opcode %d, Operands %v\n", vm.pc, instruction.Opcode, instruction.Operands)
		vm.Print()

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
			strLenBytes, ok := instruction.Operands[0].([]byte)
			if !ok || len(strLenBytes) != 4 {
				return fmt.Errorf("Invalid or missing length for string in PUSH_STRING instruction")
			}
			strLen := int(binary.LittleEndian.Uint32(strLenBytes))

			strBytes, ok := instruction.Operands[1].([]byte)
			if !ok || len(strBytes) != strLen {
				return fmt.Errorf("Invalid or missing string in PUSH_STRING instruction")
			}
			strVal := string(strBytes)
			vm.stack = append(vm.stack, strVal)
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
		case compiler.MUL:
			if len(vm.stack) < 2 {
				return fmt.Errorf("MUL instruction requires at least 2 values on the stack")
			}
			operand2, ok1 := vm.stack[len(vm.stack)-1].(float64)
			operand1, ok2 := vm.stack[len(vm.stack)-2].(float64)
			if !ok1 || !ok2 {
				return fmt.Errorf("MUL instruction requires float operands")
			}
			result := operand1 * operand2
			vm.stack = vm.stack[:len(vm.stack)-2]
			vm.stack = append(vm.stack, result)
			fmt.Printf("Stack after MUL: %v\n", vm.stack)
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
			//fmt.Printf(value)
			fmt.Println()
			vmPrint(value)
			fmt.Println()
			vm.stack = vm.stack[:len(vm.stack)-1]
		case compiler.DEFINE_VARIABLE:
			if len(instruction.Operands) < 1 {
				return fmt.Errorf("DEFINE_VARIABLE instruction requires a variable name as operand")
			}

			varBytes, ok := instruction.Operands[1].([]byte)
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
		case compiler.CREATE_LAMBDA:
			if len(instruction.Operands) < 4 {
				return fmt.Errorf("CREATE_LAMBDA instruction requires at least 4 operands")
			}

			startAddressBytes, ok := instruction.Operands[0].([]byte)
			if !ok {
				return fmt.Errorf("Invalid operand type for lambda start address")
			}
			startAddress := int(binary.LittleEndian.Uint32(startAddressBytes))

			endAddressBytes, ok := instruction.Operands[1].([]byte)
			if !ok {
				return fmt.Errorf("Invalid operand type for lambda end address")
			}
			endAddress := int(binary.LittleEndian.Uint32(endAddressBytes))

			var start int
			if startIdx, ok := vm.offsetMap[startAddress]; ok {
				start = startIdx
			} else {
				fmt.Printf("start index: %v\n", startIdx)
				return fmt.Errorf("Invalid start address for lambda")
			}

			var end int
			if endIdx, ok := vm.offsetMap[endAddress]; ok {
				end = endIdx
			} else {
				return fmt.Errorf("Invalid end address for lambda")
			}

			paramCountBytes, ok := instruction.Operands[2].([]byte)
			if !ok {
				return fmt.Errorf("Invalid operand type for lambda parameter count")
			}
			paramCount := int(binary.LittleEndian.Uint32(paramCountBytes))
			var paramNames []string
			ii := 0

			for ii < paramCount {
				paramNameLenBytes, ok := instruction.Operands[3+ii*2].([]byte)
				if !ok || len(paramNameLenBytes) != 4 {
					return fmt.Errorf("Invalid or missing length for param name in CREATE_LAMBDA instruction")
				}
				paramNameLen := int(binary.LittleEndian.Uint32(paramNameLenBytes))

				paramNameBytes, ok := instruction.Operands[4+ii*2].([]byte)
				if !ok || len(paramNameBytes) != paramNameLen {
					return fmt.Errorf("Invalid or missing param name in CREATE_LAMBDA instruction")
				}
				paramName := string(paramNameBytes)
				paramNames = append(paramNames, paramName)
				ii++
			}
			//fmt.Printf("----lambda params: %v\n", paramNames)
			iii := 4 + (ii-1)*2 + 1

			capturedVarCountBytes, ok := instruction.Operands[iii].([]byte)
			if !ok {
				return fmt.Errorf("Invalid operand type for number of captured variables")
			}
			capturedVarCount := int(binary.LittleEndian.Uint32(capturedVarCountBytes))

			var capturedVars []string

			for i := 0; i < capturedVarCount; i++ {
				varNameLenBytes, ok := instruction.Operands[(iii+1)+i*2].([]byte)
				if !ok || len(varNameLenBytes) != 4 {
					return fmt.Errorf("Invalid or missing length for captured var name in CREATE_LAMBDA instruction")
				}
				varNameLen := int(binary.LittleEndian.Uint32(varNameLenBytes))

				varNameBytes, ok := instruction.Operands[(iii+2)+i*2].([]byte)
				if !ok || len(varNameBytes) != varNameLen {
					return fmt.Errorf("Invalid or missing captured var name in CREATE_LAMBDA instruction")
				}
				varName := string(varNameBytes)
				capturedVars = append(capturedVars, varName)
			}

			lambdaSymbolTable := NewRuntimeSymbolTable(nil)

			currentSymbolTable := vm.symbolTableStack[len(vm.symbolTableStack)-1]
			for _, varName := range capturedVars {
				if value, exists := currentSymbolTable.Get(varName); exists {
					lambdaSymbolTable.Set(varName, value)
				} else {
					return fmt.Errorf("Captured variable not found in the current scope: %s", varName)
				}
			}

			lambdaFunction := &LambdaFunction{
				StartAddress: start,
				EndAddress:   end,
				SymbolTable:  lambdaSymbolTable,
				ParamCount:   paramCount,
				ParamNames:   paramNames,
				CapturedVars: capturedVars,
			}

			vm.push(lambdaFunction)
			//fmt.Printf("current stack after pushing lambda: %v\n", vm.stack)
			//fmt.Printf("Lambda created with start address %d and end address %d\n", startAddress, endAddress)
			//lambdaFunction.Print("----")
		case compiler.CALL_LAMBDA:
			numArgsBytes, ok := instruction.Operands[0].([]byte)
			if !ok {
				return fmt.Errorf("Invalid operand for CALL_LAMBDA instruction")
			}
			numArgs := int(binary.LittleEndian.Uint32(numArgsBytes))
			//fmt.Printf("number of args for lambda: %v\n", numArgs)
			//fmt.Printf("current vm stack before popping lambda args: %v\n", vm.stack)
			args := make([]interface{}, numArgs)
			for i := numArgs - 1; i >= 0; i-- {
				arg, err := vm.pop()
				if err != nil {
					return err
				}
				args[i] = arg
			}

			//fmt.Printf("current vm stack before popping lambda function: %v\n", vm.stack)
			popped, err1 := vm.pop()
			lambdaFunc, ok2 := popped.(*LambdaFunction)
			if (err1 != nil) || !ok2 {
				return fmt.Errorf("Expected a lambda function on the stack: %v\n", err1)
			}
			//fmt.Printf("popped lambda: %v\n", lambdaFunc)

			// captured vars???
			lambdaSymbolTable := NewRuntimeSymbolTable(lambdaFunc.SymbolTable)
			for i, paramName := range lambdaFunc.ParamNames {
				lambdaSymbolTable.Set(paramName, args[i])
			}
			//fmt.Printf("lambda symbol table set: \n")
			//lambdaSymbolTable.Print("----")

			//currentSymbolTable := vm.symbolTableStack[len(vm.symbolTableStack)-1]

			returnAddress := vm.pc

			vm.symbolTableStack = append(vm.symbolTableStack, lambdaSymbolTable)

			//fmt.Printf("----lambda start: %v\tlambda end: %v\n", lambdaFunc.StartAddress, lambdaFunc.EndAddress)
			err := vm.Run(lambdaFunc.StartAddress, lambdaFunc.EndAddress)
			if err != nil {
				return err
			}

			returnValue := vm.stack[len(vm.stack)-1]

			vm.pc = returnAddress
			vm.symbolTableStack = vm.symbolTableStack[:len(vm.symbolTableStack)-1]

			vm.push(returnValue)

			//return nil
		case compiler.PUSH_VARIABLE:
			varNameLenBytes, ok := instruction.Operands[0].([]byte)
			if !ok || len(varNameLenBytes) != 4 {
				return fmt.Errorf("Invalid or missing length for variable name in PUSH_VARIABLE instruction")
			}
			varNameLen := int(binary.LittleEndian.Uint32(varNameLenBytes))

			varNameBytes, ok := instruction.Operands[1].([]byte)
			if !ok || len(varNameBytes) != varNameLen {
				return fmt.Errorf("Invalid or missing variable name in PUSH_VARIABLE instruction")
			}
			varName := string(varNameBytes)

			var value interface{}
			found := false
			for i := len(vm.symbolTableStack) - 1; i >= 0; i-- {
				symbolTable := vm.symbolTableStack[i]
				if val, exists := symbolTable.symbols[varName]; exists {
					value = val
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("Variable not found: %s", varName)
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
			funcNameLenBytes, ok := instruction.Operands[0].([]byte)
			if !ok || len(funcNameLenBytes) != 4 {
				return fmt.Errorf("Invalid length for function name length in CALL_FUNCTION instruction")
			}

			funcNameLen := int(binary.LittleEndian.Uint32(funcNameLenBytes))

			funcNameBytes, ok := instruction.Operands[1].([]byte)
			if !ok || len(funcNameBytes) != funcNameLen {
				return fmt.Errorf("Invalid or missing function name in CALL_FUNCTION instruction")
			}

			funcName := string(funcNameBytes)
			fmt.Printf("CALL_FUNCTION for %s\n", funcName)

			functionMetadata, ok := vm.functions[funcName]
			if !ok {
				return fmt.Errorf("Function %s not defined", funcName)
			}

			argCount := functionMetadata.ParamCount
			if len(vm.stack) < argCount {
				return fmt.Errorf("Not enough arguments on stack for function %s", funcName)
			}

			args := make([]interface{}, argCount)
			for i := argCount - 1; i >= 0; i-- {
				arg, err := vm.pop()
				if err != nil {
					return err
				}
				args[i] = arg
			}

			newSymbolTable := NewRuntimeSymbolTable(vm.symbolTableStack[len(vm.symbolTableStack)-1])
			for i, paramName := range functionMetadata.ParamNames {
				newSymbolTable.Set(paramName, args[i])
			}
			vm.callStack = append(vm.callStack, CallStackEntry{returnAddress: vm.pc, symbolTable: vm.symbolTableStack[len(vm.symbolTableStack)-1]})
			vm.symbolTableStack = append(vm.symbolTableStack, newSymbolTable)
			vm.pc = functionMetadata.StartAddress

			fmt.Println("Jumping to function start address:", vm.pc)
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

			vm.push(returnValue)

			fmt.Printf("Returning to address %d with value %v\n", vm.pc, returnValue)
			continue
		case compiler.MAP:
			numArgs := int(binary.LittleEndian.Uint32(instruction.Operands[0].([]byte)))

			args := make([]interface{}, numArgs)
			for i := numArgs - 1; i >= 0; i-- {
				arg, err := vm.pop()
				if err != nil {
					return fmt.Errorf("error executing MAP: %v", err)
				}
				args[i] = arg
			}

			poppedLambda, err := vm.pop()
			if err != nil {
				return fmt.Errorf("error executing MAP: %v", err)
			}
			lambdaFunc, ok := poppedLambda.(*LambdaFunction)
			if !ok {
				return fmt.Errorf("error executing MAP: expected a lambda function")
			}

			results := make([]interface{}, numArgs)
			for i, arg := range args {
				result, err := vm.executeLambda(lambdaFunc, []interface{}{arg})
				if err != nil {
					return fmt.Errorf("error executing MAP with lambda: %v", err)
				}
				results[i] = result
			}

			vm.push(results)

			//return nil
		case compiler.FILTER:
			numArgsBytes, ok := instruction.Operands[0].([]byte)
			if !ok {
				return fmt.Errorf("Invalid operand for FILTER instruction")
			}
			numArgs := int(binary.LittleEndian.Uint32(numArgsBytes))

			var args []interface{}
			for i := 0; i < numArgs; i++ {
				arg, err := vm.pop()
				if err != nil {
					return err
				}
				args = append(args, arg)
			}

			poppedLambda, err := vm.pop()
			if err != nil {
				return fmt.Errorf("error executing FILTER: %v", err)
			}
			lambdaFunc, ok := poppedLambda.(*LambdaFunction)
			if !ok {
				return fmt.Errorf("error executing FILTER: expected a lambda function")
			}

			var filteredResults []interface{}
			for i := len(args) - 1; i >= 0; i-- {
				result, err := vm.executeLambda(lambdaFunc, []interface{}{args[i]})
				if err != nil {
					return err
				}

				if resultBool, ok := result.(bool); ok && resultBool {
					filteredResults = append(filteredResults, args[i])
				}
			}

			vm.push(filteredResults)
			//return nil
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

	fmt.Println()
	fmt.Printf("All instructions executed. Last executed instruction PC: %v\n", vm.pc)
	fmt.Printf("Exiting VM...\n")
	fmt.Println()
	return nil
}

func (vm *VM) executeLambda(lambdaFunc *LambdaFunction, args []interface{}) (interface{}, error) {
	if len(args) != lambdaFunc.ParamCount {
		return nil, fmt.Errorf("lambda function expected %d arguments, got %d", lambdaFunc.ParamCount, len(args))
	}

	savedPC := vm.pc

	lambdaSymbolTable := NewRuntimeSymbolTable(lambdaFunc.SymbolTable)
	for i, paramName := range lambdaFunc.ParamNames {
		lambdaSymbolTable.Set(paramName, args[i])
	}
	vm.symbolTableStack = append(vm.symbolTableStack, lambdaSymbolTable)

	vm.pc = lambdaFunc.StartAddress
	err := vm.Run(lambdaFunc.StartAddress, lambdaFunc.EndAddress)
	if err != nil {
		return nil, err
	}

	var returnValue interface{}
	if len(vm.stack) > 0 {
		returnValue = vm.stack[len(vm.stack)-1]
		vm.stack = vm.stack[:len(vm.stack)-1]
	}

	vm.pc = savedPC
	vm.symbolTableStack = vm.symbolTableStack[:len(vm.symbolTableStack)-1]

	return returnValue, nil
}

func (vm *VM) jumpToOpcode(targetOpcode compiler.Opcode) {
	for i := vm.pc + 1; i < len(vm.code); i++ {
		if vm.code[i].Opcode == targetOpcode {
			vm.pc = i
			return
		}
	}
}

func vmPrint(v interface{}) {
	vStr := fmt.Sprintf("%v", v)
	boxWidth := len(vStr) + 4
	topBorder := strings.Repeat("-", boxWidth)
	middle := fmt.Sprintf("| %s |", vStr)
	fmt.Println(topBorder)
	fmt.Println(middle)
	fmt.Println(topBorder)
}
