package newstuff

import "unsafe"
import "gowasm/rt/wasm"

type Point struct {
	x int32
	y int32
}

//wasm:invoke (invoke "Peek32" (i32.const 0))
func Peek32(addr uintptr) int32 {
	u1 := unsafe.Pointer(addr)
	p := (*int32)(u1)
	i32 := *p
	return i32
}

//wasm:invoke (invoke "TestPeek32")
func TestPeek32() {
	p := &Point{}
	p.x = int32(17)
	xp := &p.x
	u1 := unsafe.Pointer(xp)
	u := uintptr(u1)
	i32 := Peek32(u)
	wasm.Print_int32(i32)
}

//wasm:invoke (invoke "DumpMemory" (i32.const 0) (i32.const 100))
func DumpMemory(start, end uintptr) {
	for i := start; i < end; i = i + 4 {
		wasm.Print_int32(Peek32(i))
	}
}
