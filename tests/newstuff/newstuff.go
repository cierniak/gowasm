package newstuff

import "gowasm/rt/wasm"

var global32 int32 = 15

//wasm:invoke (invoke "TestGlobalVar")
func TestGlobalVar() {
	wasm.Print_int32(global32)
}
