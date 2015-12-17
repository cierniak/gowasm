package newstuff

//import "gowasm/rt/wasm"

//wasm:assert_return (invoke "TestArray3" (i32.const 13) (i32.const 15)) (i32.const 15)
func TestArray3(m, n int8) int8 {
	a := [...]int8{m, n}
	a[0] = int8(11)
	return a[1]
}
