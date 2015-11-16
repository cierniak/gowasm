package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

var dumpAST bool
var verbose bool
var outFile string

func initFlags() {
	flag.BoolVar(&dumpAST, "d", false, "print the Go AST to stdout")
	flag.BoolVar(&verbose, "v", false, "print out extra information")
	flag.StringVar(&outFile, "o", "out.wast", "output file")
	flag.Parse()
}

type FormattingWriter interface {
	Printf(format string, a ...interface{}) (n int, err error)
	PrintfIndent(indent int, format string, a ...interface{}) (n int, err error)
}

type FormattingWriterImpl struct {
	b bytes.Buffer
}

func (w *FormattingWriterImpl) Printf(format string, a ...interface{}) (n int, err error) {
	s := fmt.Sprintf(format, a...)
	return (&w.b).Write([]byte(s))
}

const indentPattern = "  "

func (w *FormattingWriterImpl) PrintfIndent(indent int, format string, a ...interface{}) (n int, err error) {
	indentString := strings.Repeat(indentPattern, indent)
	s := fmt.Sprintf(format, a...)
	return (&w.b).Write([]byte(indentString + s))
}

func (w *FormattingWriterImpl) WriteToFile(name string) (int, error) {
	f, err := os.Create(name)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return f.Write(w.b.Bytes())
}

func Compile(fileName string, writer FormattingWriter, m WasmModuleLinker) {
	fmt.Printf("Compiling file '%s'\n", fileName)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fileName, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	if dumpAST {
		ast.Print(fset, f)
	}

	err = m.addAstFile(f, fset)
	if err != nil {
		panic(err)
	}
}

func main() {
	initFlags()
	writer := &FormattingWriterImpl{}
	m := NewWasmModuleLinker()
	for _, f := range flag.Args() {
		Compile(f, writer, m)
	}

	if err := m.finalize(); err != nil {
		panic(err)
	}

	m.print(writer)

	if verbose {
		fmt.Printf("--- begin WASM output\n%s\n--- end WASM output\n", writer.b.String())
	}

	_, err := writer.WriteToFile(outFile)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Output written to '%s'\n", outFile)
}
