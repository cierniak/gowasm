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

//wasm:assert_return (invoke "TestFloat2") (f64.const 17.0)
func TestFloat2() float64 {
	a := float64(20.0)
	return a - float64(3.0)
}

//wasm:assert_return (invoke "TestFloat3") (f32.const 4.0)
func TestFloat3() float32 {
	a := float32(20.0)
	b := addFloat32(float32(3.0), float32(2.0))
	if a > b {
		return a / b
	} else {
		return 2.0
	}
}
