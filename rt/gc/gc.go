package gc

var freePointer int32

func Alloc(size, align int32) int32 {
	mem := freePointer
	freePointer = freePointer + size
	return mem
}

//wasm:assert_return (invoke "Align" (i32.const 9) (i32.const 4)) (i32.const 12)
//wasm:assert_return (invoke "Align" (i32.const 16) (i32.const 4)) (i32.const 16)
//wasm:assert_return (invoke "Align" (i32.const 20) (i32.const 8)) (i32.const 24)
//wasm:assert_return (invoke "Align" (i32.const 21) (i32.const 1)) (i32.const 21)
func Align(addr, alignment int32) int32 {
	addr = addr + (alignment - 1)
	mask := ^(alignment - 1)
	addr = addr & mask
	return addr
}
