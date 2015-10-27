package i32

//wasm:assert_return (invoke "Add" (i32.const 1) (i32.const 5)) (i32.const 6)
func Add(a, b int32) int32 {
	c := a + b
	return c
}

func Two() int32 {
	return 2
}
