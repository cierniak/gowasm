package mem

//wasm:assert_return (invoke "F" (i32.const 6)) (i32.const 6)
func F(a int32) int32 {
	return 6
}
