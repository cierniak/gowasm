package main

import (
	"fmt"
)

type WasmMemory struct {
	size           int
	nextStaticAddr int
	content        []byte
}

func createMemory(size int) *WasmMemory {
	memory := &WasmMemory{
		size:           size,
		nextStaticAddr: 4,
		content:        make([]byte, 0, size),
	}
	return memory
}

func (memory *WasmMemory) allocGlobal(size, align int) int {
	addr := memory.nextStaticAddr + (align - 1)
	mask := ^(align - 1)
	addr = addr & mask
	nextAddr := addr + size
	if nextAddr > cap(memory.content) {
		panic(fmt.Sprintf("Out of static memory: %d", nextAddr))
	}
	if nextAddr > len(memory.content) {
		memory.content = memory.content[:nextAddr]
	}
	memory.nextStaticAddr = nextAddr
	return addr
}

func (memory *WasmMemory) writeInt32(addr int, val int32) {
	for i := 0; i < 4; i++ {
		b := val & 0xff
		memory.content[addr+i] = byte(b)
		val >>= 8
	}
}

func (memory *WasmMemory) writeBytes(addr int, bytes []byte) {
	for i, b := range bytes {
		memory.content[addr+i] = b
	}
}

func (memory *WasmMemory) print(writer FormattingWriter) {
	indent := 1
	writer.Printf("\n")
	writer.PrintfIndent(indent, "(memory %d\n", memory.size)

	// Static memory segment
	writer.PrintfIndent(indent+1, "(segment 0 \"")
	for _, b := range memory.content {
		switch {
		default:
			writer.Printf("\\%02x", b)
		case ' ' <= b && b <= '}':
			writer.Printf("%c", b)
		}
	}
	writer.Printf("\") ;; static memory\n")

	// Heap segment
	writer.PrintfIndent(indent+1, "(segment %d \"", len(memory.content))
	writer.Printf("\") ;; heap\n")

	writer.PrintfIndent(indent, ")\n")
}
