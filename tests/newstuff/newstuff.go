package newstuff

//import "gowasm/rt/wasm"
//import "gowasm/rt/gc"
//import "unsafe"

//wasm:assert_return (invoke "TestArray5") (i32.const 101)
func TestArray5() byte {
	a := [...]byte{'h', 'e', 'l', 'l', 'o', 0}
	return a[1]
}
