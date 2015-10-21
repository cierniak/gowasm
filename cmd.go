package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

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

	ast.Print(fset, f)

	m, err := parseAstFile(f, fset)
	if err != nil {
		panic(err)
	}
	m.print(writer)
}

func main() {
	fmt.Printf("Go WASM, arg=%v\n", os.Args)
	w := &FormattingWriterImpl{}
	for i, f := range os.Args[1:] {
		fmt.Printf("Compiling file #%d: '%s'\n", i, f)
		Compile(f, w)
	}
	fmt.Printf("WASM output:\n%s\nEOF\n", w.b.String())

	fileName := "out.wast"
	_, err := w.WriteToFile(fileName)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Output written to '%s'\n", fileName)
}
