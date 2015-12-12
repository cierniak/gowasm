package newstuff

//import "gowasm/rt/wasm"

func addFloat32(a, b float32) float32 {
	return a + b
}

//wasm:assert_return (invoke "TestBitwise1" (i64.const 9) (i64.const 3)) (i64.const 721162)
func TestBitwise1(a, b int64) int64 {
	r := (a | b) << 8
	r = (r | (a & b)) << 8
	r = r | (a ^ b)
	return r
}

//wasm:assert_return (invoke "TestBitwise2") (i64.const 15)
func TestBitwise2() int64 {
	a := int64(0 - 1)
	a = a >> 60
	b := uint64(a)
	b = b >> 60
	return int64(b)
}
