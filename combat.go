package main

var (
	validWeapons = []int{1, 2, 3, 4, 5, 6, 7, 8, 10, 12, 14, 15, 16, 20, 40}
)

//NetworkWeapon holds a player's current weapon according to the network
type NetworkWeapon struct {
	FightState  FightState
	WeaponType  WeaponType
	Projectiles []Projectile
}

//FightState is the fighting state of a player
type FightState byte

//WeaponType is the type of weapon being used
type WeaponType byte

//Projectile holds a projectile from a weapon
type Projectile struct {
	Shoot         Vector2
	ShootPosition Vector2
	SyncIndex     uint16
}

//DamageType is the type of damage being taken
type DamageType byte

const (
	damageTypePunch DamageType = iota
	damageTypeLocalDamage
	damageTypeOther
)

func (damageType DamageType) String() string {
	switch damageType {
	case damageTypePunch:
		return "Punch"
	case damageTypeLocalDamage:
		return "LocalDamage"
	case damageTypeOther:
		return "Other"
	default:
		return "Unknown"
	}
}
