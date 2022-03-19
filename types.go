package main

import (
	"fmt"
)

//Vector2 holds a 2D coordinate
type Vector2 struct {
	X, Y float32
}

func (vector2 Vector2) String() string {
	return fmt.Sprintf("X:%.2f Y:%.2f", vector2.X, vector2.Y)
}

func (vector2 Vector2) Vector3() Vector3 {
	return Vector3{
		X: vector2.X,
		Y: vector2.Y,
		Z: 0,
	}
}

//Vector3 holds a 3D coordinate
type Vector3 struct {
	X, Y, Z float32
}

func (vector3 Vector3) String() string {
	return fmt.Sprintf("X:%.2f Y:%.2f Z:%.2f", vector3.X, vector3.Y, vector3.Z)
}

func (vector3 Vector3) Vector2() Vector2 {
	return Vector2{
		X: vector3.X,
		Y: vector3.Y,
	}
}
