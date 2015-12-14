package newstuff

//import "gowasm/rt/wasm"

//wasm:assert_return (invoke "TestBitwise3") (i64.const -6)
func TestBitwise3() int64 {
	a := int64(5)
	return ^a
}
