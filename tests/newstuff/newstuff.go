package newstuff

//import "gowasm/rt/wasm"

type IntFunc func() int
type IntFunc2 func(int, int) int

func twelve() int {
	return 12
}

func sum(a, b int) int {
	return a + b
}

func max(a, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

func testIntFunc2(f IntFunc2, a, b int) int {
	return f(a, b)
}

//wasm:assert_return (invoke "Test1") (i32.const 12)
func Test1() int {
	var f IntFunc
	f = twelve
	return f()
}

//wasm:assert_return (invoke "Test2") (i32.const 17)
func Test2() int {
	r := testIntFunc2(sum, 10, 3)
	r = r + testIntFunc2(max, 2, 4)
	return r
}
