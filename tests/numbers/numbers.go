package numbers

func addFloat32(a, b float32) float32 {
	return a + b
}

//wasm:assert_return (invoke "TestFloat1") (f32.const 20.0)
func TestFloat1() float32 {
	var a float32
	a = float32(2.0)
	b := addFloat32(float32(4.0), float32(5.0))
	return a * (b + 1.0)
}
