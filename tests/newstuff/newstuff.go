package newstuff

import (
	"gowasm/rt/gc"
)

type Point struct {
	x int32
	y int32
}

func newPoint(x, y int32) *Point {
	p := &Point{}
	p.x = x
	p.y = y
	return p
}

func G(x, y int32) int32 {
	p := newPoint(x, y)
	return p.x + p.y
}
