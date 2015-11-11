package gc

var freePointer int32

//wasm:assert_return (invoke "alloc" (i32.const 128) (i32.const 64)) (i32.const 128)
//wasm:assert_return (invoke "alloc" (i32.const 64) (i32.const 32)) (i32.const 128)
func Alloc(size, align int32) int32 {
	mem := freePointer
	freePointer = freePointer + size
	return mem
}
