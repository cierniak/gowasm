package mem

type Point struct {
	x int32
	y int32
}

//wasm:assert_return (invoke "F" (i32.const 6)) (i32.const 0)
func F(a int32) int32 {
	p := &Point{}
	return p.x
}
