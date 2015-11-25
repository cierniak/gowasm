package newstuff

import "gowasm/rt/wasm"

var global32 int32 = 15

//wasm:assert_return (invoke "TestGlobalVar") (i32.const 15)
func TestGlobalVar() int32 {
	wasm.Print_int32(global32)
	return global32
}
