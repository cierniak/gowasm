package i32

//wasm:assert_return (invoke "Add" (i32.const 1) (i32.const 5)) (i32.const 6)
func Add(a, b int32) int32 {
	c := a + b
	return c
}

//wasm:assert_return (invoke "Expr1" (i32.const 10) (i32.const 3)) (i32.const 14)
func Expr1(a, b int32) int32 {
	return 2 * (a - b)
}

func two() int32 {
	return 2
}
