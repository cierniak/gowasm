package fac

import "gowasm/rt/wasm"

//wasm:assert_return (invoke "Add" (i64.const 13) (i64.const 200)) (i64.const 213)
func Add(a, b int64) int64 {
	wasm.Print_int64(a)
	return a + b
}
