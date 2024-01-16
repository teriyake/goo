package main

import (
	"fmt"
	"teriyake/goo/lexer"
	"teriyake/goo/parser"
	"teriyake/goo/compiler"
	"teriyake/goo/vm"
)

func main() {
	//gooCode := "(let x:int 3) (print (- x 1))"
	gooCode := "(def factorial (x:int) (if (= x 0) (ret 1) else (ret (* x factorial(- x 1))))) (print (factorial(5)))"
	//gooCode := "(def double (x: int) (ret (* x 2))) (print (double(7)))"
	//gooCode := "(def add (x:int y:int) (ret (+ x y))) (print (add(1 2)))"
	//gooCode := "(let x:int -2) (if (< x 3) (if (> x 1) (print 'x is greater than 1 and less than 3')) else (if (= x -2) (print 'x equals -2') else (print 'x is greater than 3')))"
	//gooCode := "(def hello (a b) (print a) (print b) (print 'hi') (print '123') (print 123)) (hello 2 4)"
	//gooCode := "(def add_num (a b) (ret (+ a b))) (add_num 3 5)"
	//gooCode := "(let x 1) (print x)"
	//gooCode := "(let x 10) (let x 11)"
	fmt.Printf("Input: %v\n", gooCode)
	fmt.Println()
	lexer := lexer.NewLexer(gooCode)
	par := parser.NewParser(lexer)
	ast, err := par.Parse()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	} else {
		fmt.Printf("AST: %#v\n", ast)
		fmt.Println()
	}

	comp := compiler.NewCompiler()
	bytecodeInstructions, err := comp.CompileAST(ast)
	if err != nil {
		fmt.Printf("Error compiling AST: %s\n", err)
		return
	}
	//fmt.Printf("Generated Bytecode: %v\n", bytecode)
	//fmt.Printf("Generated Bytecode Instructions: %v\n", bytecodeInstructions)
	for i, b := range bytecodeInstructions {
		fmt.Printf("Pos: %v\tOpcode: %v\tOperands: %v\n", i, compiler.OpcodeToString(b.Opcode), b.Operands)
	}
	fmt.Println()

	virtualMachine := vm.NewVM(bytecodeInstructions)
	fmt.Printf("Initial VM State: \n")
	virtualMachine.Print()
	fmt.Println()

	err = virtualMachine.Run()
	if err != nil {
		fmt.Printf("Error executing Goo code: %s\n", err)
	}
	fmt.Printf("Final VM State: \n")
	virtualMachine.Print()
	fmt.Println()
}
