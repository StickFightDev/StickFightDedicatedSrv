package main

type vector3 struct {
	X, Y, Z float32
}

type quaternion struct {
	X, Y, Z, W float32
}

type damageType byte

func (dt damageType) String() string {
	if dt == damageTypePunch {
		return "Punch"
	} else if dt == damageTypeLocalDamage {
		return "LocalDamage"
	} else if dt == damageTypeOther {
		return "Other"
	} else {
		return "Unknown"
	}
}

const (
	damageTypePunch damageType = iota
	damageTypeLocalDamage
	damageTypeOther
)
