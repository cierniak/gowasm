package newstuff

//import "unsafe"
import "gowasm/rt/wasm"

/*
//wasm:invoke (invoke "DumpMemory" (i32.const 0) (i32.const 10))
func DumpMemory(start, end int32) {
	for i := start; i < end; i = i + 4 {
		wasm.Print_int32(i)
	}
}
*/
//wasm:invoke (invoke "Peek32" (i32.const 0))
func Peek32(addr int32) {
	var n int32
	n = addr
	wasm.Print_int32(n)
}
