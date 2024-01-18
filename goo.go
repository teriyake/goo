package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"teriyake/goo/compiler"
	"teriyake/goo/lexer"
	"teriyake/goo/parser"
	"teriyake/goo/vm"
)

type FileWriter struct {
	file *os.File
}

func NewFileWriter(fileName string) (*FileWriter, error) {
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	return &FileWriter{file}, nil
}

func (fw *FileWriter) Write(p []byte) (n int, err error) {
	return fw.file.Write(p)
}

func main() {
	debugMode := flag.Bool("debug", false, "when enabled, the compiler and vm debug outputs will be piped to a log file")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Println("./goo [-debug path/to/log.log] path/to/src.goo")
		fmt.Println("\nFlags:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("Usage: ./goo [-debug path/to/log.log] path/to/src.goo")
		os.Exit(1)
	}

	var logFilePath, srcFilePath string
	if *debugMode {
		if flag.NArg() < 2 {
			fmt.Println("Usage: ./goo [-debug path/to/log.log] path/to/src.goo")
			os.Exit(1)
		}
		logFilePath = flag.Arg(0)
		srcFilePath = flag.Arg(1)
	} else {
		srcFilePath = flag.Arg(0)
	}

	srcCode, err := ioutil.ReadFile(srcFilePath)
	if err != nil {
		fmt.Printf("Error reading source file %s: %s\n", srcFilePath, err)
		os.Exit(1)
	}

	gooCode := string(srcCode)

	if *debugMode && logFilePath != "" {
		fw, err := NewFileWriter(logFilePath)
		if err != nil {
			fmt.Printf("Error opening log file %s: %s\n", logFilePath, err)
			os.Exit(1)
		}
		defer fw.file.Close()

		originalStdout := os.Stdout
		os.Stdout = fw.file
		defer func() { os.Stdout = originalStdout }()
	}

	if *debugMode {
		fmt.Printf("DEBUG MODE ENABLED\n")
		fmt.Println()
		fmt.Printf("Input: %v\n", gooCode)
		fmt.Println()
	}

	lexer := lexer.NewLexer(gooCode)
	par := parser.NewParser(lexer)
	ast, err := par.Parse()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	} else if *debugMode {
		fmt.Printf("AST: %#v\n", ast)
		fmt.Println()
	}

	comp := compiler.NewCompiler(debugMode)
	bytecodeInstructions, offsetMap, err := comp.CompileAST(ast)
	if err != nil {
		fmt.Printf("Error compiling AST: %s\n", err)
		return
	}
	if *debugMode {
		for i, b := range bytecodeInstructions {
			fmt.Printf("Pos: %v\tOpcode: %v %v\tOperands: %v\n", i, b.Opcode, compiler.OpcodeToString(b.Opcode), b.Operands)
		}
		fmt.Println()
	}

	virtualMachine := vm.NewVM(bytecodeInstructions, offsetMap, debugMode)
	if *debugMode {
		fmt.Printf("Initial VM State: \n")
		virtualMachine.Print()
		fmt.Println()
	}

	err = virtualMachine.Run()
	if err != nil {
		fmt.Printf("Error executing Goo code: %s\n", err)
	}
	if *debugMode {
		fmt.Printf("Final VM State: \n")
		virtualMachine.Print()
		fmt.Println()
	}
}
