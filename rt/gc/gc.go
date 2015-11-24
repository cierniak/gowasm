package gc

var freePointer int32

//wasm:assert_return (invoke "Alloc" (i32.const 8) (i32.const 4)) (i32.const 0)
//wasm:assert_return (invoke "Alloc" (i32.const 4) (i32.const 4)) (i32.const 8)
func Alloc(size, align int32) int32 {
	mem := freePointer
	freePointer = freePointer + size
	return mem
}
