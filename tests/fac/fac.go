package fac

import "gowasm/rt/wasm"

//wasm:assert_return (invoke "Fact" (i64.const 3)) (i64.const 3)
//wasm:assert_return (invoke "Fact" (i64.const 0)) (i64.const 13)
func Fact(n int64) int64 {
	if n == 0 {
		return 13
	}
	wasm.Print_int64(n)
	return n
}
