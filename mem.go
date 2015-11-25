package main

import (
	"fmt"
)

type WasmMemory struct {
	size          int
	globalVarAddr int
	content       []byte
}

func createMemory(size int) *WasmMemory {
	memory := &WasmMemory{
		size:          size,
		globalVarAddr: 4,
		content:       make([]byte, 0, size),
	}
	return memory
}

func (memory *WasmMemory) addGlobal(addr, size int) {
	limit := addr + size
	if limit > cap(memory.content) {
		panic(fmt.Sprintf("Address %d is too large", addr))
	}
	if limit > len(memory.content) {
		memory.content = memory.content[:limit]
	}
}

func (memory *WasmMemory) writeInt32(addr int, val int32) {
	memory.addGlobal(addr, 4)
	for i := 0; i < 4; i++ {
		b := val & 0xff
		memory.content[addr+i] = byte(b)
		val >>= 8
	}
}

func (memory *WasmMemory) print(writer FormattingWriter) {
	indent := 1
	writer.Printf("\n")
	writer.PrintfIndent(indent, "(memory %d\n", memory.size)

	// Globals segment
	writer.PrintfIndent(indent+1, "(segment 0 \"")
	for _, b := range memory.content {
		writer.Printf("\\%02x", b)
	}
	writer.Printf("\") ;; global variables\n")

	// Heap segment
	writer.PrintfIndent(indent+1, "(segment %d \"", len(memory.content))
	writer.Printf("\") ;; heap\n")

	writer.PrintfIndent(indent, ")\n")
}
