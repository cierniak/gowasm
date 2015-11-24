package newstuff

import "unsafe"
import "gowasm/rt/wasm"

type Point struct {
	x int32
	y int32
}

/*
//wasm:invoke (invoke "DumpMemory" (i32.const 0) (i32.const 10))
func DumpMemory(start, end int32) {
	for i := start; i < end; i = i + 4 {
		wasm.Print_int32(i)
	}
}
*/
//wasm:invoke (invoke "Peek32" (i32.const 0))
func Peek32(addr int32) {
	var n int32
	n = addr
	wasm.Print_int32(n)
}

//wasm:invoke (invoke "TestPeek32")
func TestPeek32() {
	p := &Point{}
	p.x = int32(17)
	xp := &p.x
	u1 := unsafe.Pointer(xp)
	u := uintptr(u1)
	i32 := int32(u)
	Peek32(i32)
}
