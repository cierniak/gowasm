package newstuff

//import "gowasm/rt/wasm"

//wasm:assert_return (invoke "TestArray1") (i32.const 6)
func TestArray1() int32 {
	var a [17]int32
	//a[0] = 13
	return a[0]
}
