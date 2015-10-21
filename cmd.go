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
	SprintPosition(pos token.Pos, fset *token.FileSet) string
}

type FormattingWriterImpl struct {
	b bytes.Buffer
}

func (w *FormattingWriterImpl) Printf(format string, a ...interface{}) (n int, err error) {
	s := fmt.Sprintf(format, a...)
	return (&w.b).Write([]byte(s))
}

func (w *FormattingWriterImpl) PrintfIndent(indent int, format string, a ...interface{}) (n int, err error) {
	indentString := strings.Repeat("  ", indent)
	s := fmt.Sprintf(format, a...)
	return (&w.b).Write([]byte(indentString + s))
}

func (w *FormattingWriterImpl) SprintPosition(pos token.Pos, fset *token.FileSet) string {
	position := fset.File(pos).PositionFor(pos, false)
	return fmt.Sprintf("[%s:%d:%d]", position.Filename, position.Line, position.Offset)
}

func (w *FormattingWriterImpl) WriteToFile(name string) (int, error) {
	f, err := os.Create(name)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return f.Write(w.b.Bytes())
}

func Compile(fileName string, writer FormattingWriter) {
	fmt.Printf("Compiling file '%s'\n", fileName)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fileName, nil, 0)
	if err != nil {
		panic(err)
	}

	if dumpAST {
		ast.Print(fset, f)
	}

	m, err := parseAstFile(f, fset)
	if err != nil {
		panic(err)
	}
	m.print(writer)
}

func main() {
	initFlags()
	w := &FormattingWriterImpl{}
	for _, f := range flag.Args() {
		Compile(f, w)
	}

	if verbose {
		fmt.Printf("--- begin WASM output\n%s\n--- end WASM output\n", w.b.String())
	}

	_, err := w.WriteToFile(outFile)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Output written to '%s'\n", outFile)
}
