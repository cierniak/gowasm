package i32

//wasm:assert_return (invoke "Add" (i32.const 1) (i32.const 5)) (i32.const 6)
func Add(a, b int32) int32 {
	c := a
	c = c + b
	return c
}

//wasm:assert_return (invoke "Inc" (i32.const 10)) (i32.const 11)
func Inc(n int32) int32 {
	n++
	return n
}

//wasm:assert_return (invoke "Expr1" (i32.const 10) (i32.const 3)) (i32.const 16)
func Expr1(a, b int32) int32 {
	return 2*(a-b) + two()
}

//wasm:assert_return (invoke "Expr2" (i32.const 100) (i32.const 20) (i32.const 5)) (i32.const 104)
func Expr2(a, b, c int32) int32 {
	return Add(a, b/c)
}

//wasm:assert_return (invoke "NestedLoop" (i32.const 5) (i32.const 7)) (i32.const 62)
func NestedLoop(a, b int32) int32 {
	sum := int32(0)
	for i := int32(0); i < 10; i++ {
		sum = sum + a
	}
	return sum + a + b
}

func two() int32 {
	return 2
}
