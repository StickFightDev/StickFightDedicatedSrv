package main

type vector3 struct {
	X, Y, Z float32
}

type quaternion struct {
	X, Y, Z, W float32
}

type damageType byte

const (
	damageTypePunch damageType = iota
	damageTypeLocalDamage
	damageTypeOther
)
