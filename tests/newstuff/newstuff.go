package newstuff

//import "gowasm/rt/wasm"
//import "gowasm/rt/gc"
//import "unsafe"

//wasm:assert_return (invoke "TestArray5") (i32.const 15)
func TestArray5() byte {
	a := [...]byte{13, 15, 17}
	return a[1]
}
