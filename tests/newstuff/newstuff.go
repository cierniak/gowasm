package newstuff

//import "gowasm/rt/wasm"
/*
type IntFunc func() int

func Twelve() int {
	return 12
}

//wasm:assert_return (invoke "Test1") (i32.const 14)
func Test1() int {
	var f IntFunc
	f = Twelve
	return f()
}
*/

//wasm:invoke (invoke "LoopTest" (i32.const 55) (i32.const 100))
func LoopTest(start, end int32) {
	for i := start; i < end; i = i + 4 {
		wasm.Print_int32(i)
	}
}
