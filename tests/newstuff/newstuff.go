package newstuff

//import "gowasm/rt/wasm"
//import "gowasm/rt/gc"
//import "unsafe"

//wasm:assert_return (invoke "TestArray6") (i32.const 65)
func TestArray6() byte {
	a := [...]byte{'h', 'e', 'l', 'l', 'o', 0}
	p := &a[2]
	*p = 65
	return a[2]
}
