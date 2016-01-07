package v8

import (
	"fmt"
)

func Puts(p *byte) int {
	fmt.Printf("%v : pointer (puts)\n", *p)
	return 0
}
