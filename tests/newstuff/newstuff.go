package newstuff

import "gowasm/rt/wasm"

//wasm:assert_return (invoke "Test1") (i32.const 14)
func Test1() int32 {
	var i32 int32
	i32 = int32(14)
	wasm.Print_int32(i32)
	return i32
}
