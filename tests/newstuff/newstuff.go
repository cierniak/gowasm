package newstuff

//import "gowasm/rt/wasm"

//wasm:assert_return (invoke "TestArray2") (i32.const 13)
func TestArray2() int32 {
	a := [...]int32{13, 15}
	return a[1]
}
