package newstuff

import "gowasm/rt/wasm"

//import "gowasm/rt/gc"
//import "unsafe"

//func readByte(p *byte) byte {
//	return *p
//}

//wasm:assert_return (invoke "TestPuts") (i32.const 0)
func TestPuts() int {
	a := [...]byte{'h', 'e', 'l', 'l', 'o', 0}
	p := &a[0]
	wasm.Puts(p)
	return 0
}
