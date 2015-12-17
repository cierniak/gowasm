package newstuff

//import "gowasm/rt/wasm"

//wasm:assert_return (invoke "TestArray3") (i32.const 15)
func TestArray3() int8 {
	a := [...]int8{13, 15}
	return a[1]
}
