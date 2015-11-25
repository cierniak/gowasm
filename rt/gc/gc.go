package gc

var freePointer int32

func Alloc(size, align int32) int32 {
	mem := freePointer
	freePointer = freePointer + size
	return mem
}
