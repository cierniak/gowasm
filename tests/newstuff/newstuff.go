package newstuff

import "unsafe"
import "gowasm/rt/wasm"

type Point struct {
	x int32
	y int32
}

//wasm:invoke (invoke "PtrConvert")
func PtrConvert() {
	p := &Point{}
	u1 := unsafe.Pointer(p)
	u := uintptr(u1)
	i32 := int32(u)
	wasm.Print_int32(i32)
}
