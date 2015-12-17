package newstuff

//import "gowasm/rt/wasm"

//wasm:assert_return (invoke "TestInt16" (i32.const 12) (i32.const 8) (i32.const 0)) (i32.const 20)
func TestInt16(a, b, c int16) uint16 {
	// TODO: truncate and sign-extend 16-bit ints
	return uint16(a + b + c)
}
