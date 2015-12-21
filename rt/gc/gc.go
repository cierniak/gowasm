package gc

import "unsafe"

var freePointer int32

func Alloc(size, align int32) int32 {
	mem := Align(freePointer, align)
	freePointer = mem + size
	return mem
}

//wasm:assert_return (invoke "Align" (i32.const 9) (i32.const 4)) (i32.const 12)
//wasm:assert_return (invoke "Align" (i32.const 16) (i32.const 4)) (i32.const 16)
//wasm:assert_return (invoke "Align" (i32.const 20) (i32.const 8)) (i32.const 24)
//wasm:assert_return (invoke "Align" (i32.const 21) (i32.const 1)) (i32.const 21)
func Align(addr, alignment int32) int32 {
	addr = addr + (alignment - 1)
	mask := ^(alignment - 1)
	addr = addr & mask
	return addr
}

func Peek8(addr uintptr) int8 {
	u1 := unsafe.Pointer(addr)
	p := (*int8)(u1)
	i8 := *p
	return i8
}

func Poke8(addr uintptr, val int8) {
	u1 := unsafe.Pointer(addr)
	p := (*int8)(u1)
	*p = val
}

func Memcpy(dst, src uintptr, n int) {
	for i := uintptr(0); i < uintptr(n); i = i + 1 {
		Poke8(uintptr(dst)+i, Peek8(uintptr(src)+i))
	}
}
