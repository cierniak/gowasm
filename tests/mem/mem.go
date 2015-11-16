package mem

import (
	"gowasm/rt/gc"
)

type Point struct {
	x int32
	y int32
}

//wasm:assert_return (invoke "R" (i32.const 16) (i32.const 8)) (i32.const 0)
//wasm:assert_return (invoke "R" (i32.const 16) (i32.const 8)) (i32.const 16)
func R(size, align int32) int32 {
	p := gc.Alloc(size, align)
	return p
}

/*
//wasm:assert_return (invoke "F" (i32.const 6)) (i32.const 0)
func F(a int32) int32 {
	p := &Point{}
	return p.x
}
*/
