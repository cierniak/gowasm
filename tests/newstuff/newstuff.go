package newstuff

//import "gowasm/rt/wasm"

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
