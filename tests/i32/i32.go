package i32

var global32 int32 = 15

//wasm:assert_return (invoke "TestGlobalVar") (i32.const 15)
func TestGlobalVar() int32 {
	return global32
}

//wasm:assert_return (invoke "Add" (i32.const 1) (i32.const 5)) (i32.const 6)
func Add(a, b int32) int32 {
	c := a
	c = c + b
	return c
}

//wasm:assert_return (invoke "Inc" (i32.const 10)) (i32.const 11)
func Inc(n int32) int32 {
	n++
	return n
}

//wasm:assert_return (invoke "Expr1" (i32.const 10) (i32.const 3)) (i32.const 16)
func Expr1(a, b int32) int32 {
	return 2*(a-b) + two()
}

//wasm:assert_return (invoke "Expr2" (i32.const 100) (i32.const 20) (i32.const 5)) (i32.const 104)
func Expr2(a, b, c int32) int32 {
	return Add(a, b/c)
}

//wasm:assert_return (invoke "NestedLoop" (i32.const 5) (i32.const 7)) (i32.const 35)
func NestedLoop(a, b int32) int32 {
	var sum int32
	sum = int32(0)
	for i := int32(0); i < a; i++ {
		for j := int32(0); j < b; j++ {
			sum = sum + 1
		}
	}
	return sum
}

func two() int32 {
	return 2
}

//wasm:assert_return (invoke "DivSigned" (i32.const 100) (i32.const 20)) (i32.const 5)
func DivSigned(a, b int32) int32 {
	return a / b
}

//wasm:assert_return (invoke "DivUnsigned" (i32.const 100) (i32.const 20)) (i32.const 5)
func DivUnsigned(a, b uint32) uint32 {
	return a / b
}

//wasm:assert_return (invoke "DistanceUnsigned" (i32.const 100) (i32.const 20)) (i32.const 80)
//wasm:assert_return (invoke "DistanceUnsigned" (i32.const 100) (i32.const 100)) (i32.const 0)
//wasm:assert_return (invoke "DistanceUnsigned" (i32.const 30) (i32.const 100)) (i32.const 70)
func DistanceUnsigned(a, b uint32) uint32 {
	if a > b {
		return a - b
	} else {
		return b - a
	}
}

//wasm:assert_return (invoke "AddUintPtr" (i32.const 5) (i32.const 3)) (i32.const 8)
func AddUintPtr(x, y uintptr) uintptr {
	return x + y
}
