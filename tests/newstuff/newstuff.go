package newstuff

//import "gowasm/rt/wasm"
import "unsafe"

/*
//wasm:invoke (invoke "Peek8" (i32.const 4))
func Peek8(addr uintptr) int8 {
	u1 := unsafe.Pointer(addr)
	p := (*int8)(u1)
	i8 := *p
	return i8
}
*/
func Poke8(addr uintptr, val int8) {
	u1 := unsafe.Pointer(addr)
	p := (*int8)(u1)
	*p = val
}

/*
//wasm:invoke (invoke "Memcpy" (i32.const 0) (i32.const 4) (i32.const 2))
func Memcpy(dst, src uintptr, n int) {
	for i := 0; i < n; i = i + 1 {
	}
}
*/
