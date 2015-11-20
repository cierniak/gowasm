package newstuff

import "unsafe"

type Point struct {
	x int32
	y int32
}

func H(x, y int32) uintptr {
	p := &Point{}
	u1 := unsafe.Pointer(p)
	u := uintptr(u1)
	return u
}
