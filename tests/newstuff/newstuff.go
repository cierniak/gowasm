package newstuff

//import "gowasm/rt/wasm"
import "gowasm/rt/gc"
import "unsafe"

//wasm:assert_return (invoke "TestArray4") (i32.const 15)
func TestArray4() int8 {
	a := [...]int8{13, 15, 17}
	b := [...]int8{3, 5, 7}
	a1 := unsafe.Pointer(&a)
	b1 := unsafe.Pointer(&b)
	gc.Memcpy(uintptr(b1), uintptr(a1), 3)
	return b[1]
}
