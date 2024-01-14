package main

import (
	"fmt"
	"teriyake/goo/lexer"
	"teriyake/goo/parser"
	"teriyake/goo/compiler"
	"teriyake/goo/vm"
)

func main() {
	gooCode := "(def abcde (? 2 2))"
	lexer := lexer.NewLexer(gooCode)
	par := parser.NewParser(lexer)
	ast, err := par.Parse()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	} else {
		fmt.Printf("%#v\n", ast)
	}
	comp := compiler.NewCompiler()
	bytecodeInstructions, err := comp.CompileAST(ast)
	if err != nil {
		fmt.Printf("Error compiling AST: %s\n", err)
		return
	}
	//fmt.Printf("Generated Bytecode: %v\n", bytecode)
	fmt.Printf("Generated Bytecode Instructions: %v\n", bytecodeInstructions)

	virtualMachine := vm.NewVM(bytecodeInstructions)
	fmt.Printf("Initial VM State:\n%+v\n", virtualMachine)

	err = virtualMachine.Run()
	if err != nil {
		fmt.Printf("Error executing Goo code: %s\n", err)
	}
	fmt.Printf("Final VM State:\n%+v\n", virtualMachine)
}
