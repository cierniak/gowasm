package newstuff

//import "gowasm/rt/wasm"

//wasm:assert_return (invoke "Test1") (i32.const 14)
func Test1() int32 {
	var i int
	i = 13
	j := 1
	return int32(i + j)
}
