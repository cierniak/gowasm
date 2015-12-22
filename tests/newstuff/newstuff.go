package newstuff

//import "gowasm/rt/wasm"
//import "gowasm/rt/gc"
//import "unsafe"

//func readByte(p *byte) byte {
//	return *p
//}

//wasm:assert_return (invoke "TestArray7") (i32.const 65)
func TestArray7() byte {
	a := [...]byte{'h', 'e', 'l', 'l', 'o', 0}
	p := &a[2]
	*p = 65
	return *p
}
