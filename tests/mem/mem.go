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

//wasm:assert_return (invoke "TestArray1") (i32.const 13)
func TestArray1() int32 {
	var a [6]int32
	a[5] = 13
	return a[5]
}

//wasm:assert_return (invoke "TestArray2") (i32.const 15)
func TestArray2() int32 {
	a := [...]int32{13, 15}
	return a[1]
}

//wasm:assert_return (invoke "TestArray3" (i32.const 13) (i32.const 15)) (i32.const 15)
func TestArray3(m, n int8) int8 {
	a := [...]int8{m, n, 'a'}
	a[0] = int8(11)
	return a[1]
}

//wasm:assert_return (invoke "TestArray4") (i32.const 15)
func TestArray4() int8 {
	a := [...]int8{13, 15, 17}
	b := [...]int8{3, 5, 7}
	a1 := unsafe.Pointer(&a)
	b1 := unsafe.Pointer(&b)
	gc.Memcpy(uintptr(b1), uintptr(a1), 3)
	return b[1]
}
