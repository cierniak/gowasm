package mem

import "gowasm/rt/gc"
import "gowasm/rt/wasm"
import "unsafe"

type Point struct {
	x int32
	y int32
}

//wasm:assert_return (invoke "R" (i32.const 16) (i32.const 8)) (i32.const 16)
func R(size, align int32) int32 {
	p1 := gc.Alloc(size, align)
	p2 := gc.Alloc(size, align)
	return p2 - p1
}

//wasm:assert_return (invoke "F" (i32.const 6)) (i32.const 6)
func F(a int32) int32 {
	p := &Point{}
	p.x = a
	return p.x
}

func newPoint(x, y int32) *Point {
	p := &Point{}
	p.x = x
	p.y = y
	return p
}

func G(x, y int32) int32 {
	p := newPoint(x, y)
	return p.x + p.y
}

//wasm:invoke (invoke "PtrConvert")
func PtrConvert() {
	p := &Point{}
	u1 := unsafe.Pointer(p)
	u := uintptr(u1)
	i32 := int32(u)
	wasm.Print_int32(i32)
}

//wasm:invoke (invoke "PtrToInt32")
func PtrToInt32() {
	p := &Point{}
	p.x = int32(17)
	xp := &p.x
	i32 := *xp
	wasm.Print_int32(i32)
}
