package newstuff

//import "unsafe"
import "gowasm/rt/wasm"

type Point struct {
	x int32
	y int32
}

//wasm:invoke (invoke "PtrConvert")
func PtrConvert() {
	p := &Point{}
	p.x = int32(17)
	xp := &p.x
	i32 := *xp
	wasm.Print_int32(i32)
}
