package fac

import "gowasm/rt/wasm"

//wasm:assert_return (invoke "Fact" (i64.const 0)) (i64.const 1)
//wasm:assert_return (invoke "Fact" (i64.const 3)) (i64.const 6)
func Fact(n int64) int64 {
	if n == 0 {
		return 1
	}
	return n * Fact(n-1)
}

//wasm:invoke (invoke "PrintAll" (i64.const 3))
func PrintAll(n int64) {
	for i := int64(0); i < 10; i++ {
		f := Fact(i)
		wasm.Print_int64(f)
	}
}
