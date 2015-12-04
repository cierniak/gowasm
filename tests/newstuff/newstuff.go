package newstuff

//import "gowasm/rt/wasm"

type IntFunc func() int
type IntFunc2 func(int, int) int

func Twelve() int {
	return 12
}

func Sum(a, b int) int {
	return a + b
}

//wasm:assert_return (invoke "Test1") (i32.const 12)
func Test1() int {
	var f IntFunc
	f = Twelve
	return f()
}

//wasm:assert_return (invoke "Test1") (i32.const 12)
func Test2() int {
	var f1 IntFunc2
	f1 = Sum
	return f1(10, 7)
}
