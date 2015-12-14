package newstuff

//import "gowasm/rt/wasm"

//wasm:assert_return (invoke "TestBitwise3") (i64.const 15)
func TestBitwise3() int64 {
	a := int64(5)
	return ^a
}
