package newstuff

//import "gowasm/rt/wasm"

func addFloat32(a, b float32) float32 {
	return a + b
}

//wasm:assert_return (invoke "TestBitwise" (i64.const 9) (i64.const 3)) (i64.const 2817)
func TestBitwise(a, b int64) int64 {
	r := (a | b) << 8
	r = r | (a & b)
	return r
}
