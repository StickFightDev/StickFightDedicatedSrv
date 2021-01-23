package main

var (
	validWeapons = []int{
		0, 6, 61, 16, 31, 1,
		60, 32, 5, 62, 10, 20,
		3, 12, 22, 24, 37, 25,
		26, 33, 64, 38, 39, 35,
		36, 40, 2, 4, 63, 21,
		14, 7, 41, 27, 19, 23,
	}
)

//Weapon is the weapon ID of a weapon
type Weapon int

const (
	weaponPistol          Weapon = 0
	weaponRevolver        Weapon = 6
	weaponDeagle          Weapon = 61
	weaponUzi             Weapon = 16
	weaponGodPistol       Weapon = 31
	weaponAK47            Weapon = 1
	weaponM16             Weapon = 60
	weaponM1              Weapon = 32
	weaponSniper          Weapon = 5
	weaponSawedOff        Weapon = 62
	weaponMilitaryShotgun Weapon = 10
	weaponBouncer         Weapon = 20
	weaponGrenadeLauncher Weapon = 3
	weaponThruster        Weapon = 12
	weaponRPG             Weapon = 22
	weaponSnakePistol     Weapon = 24
	weaponSnakeShotgun    Weapon = 37
	weaponSnakeGrenade    Weapon = 25
	weaponSnakeLauncher   Weapon = 26
	weaponSnakeMinigun    Weapon = 33
	weaponFlyingSnake     Weapon = 64
	weaponSpikeBall       Weapon = 38
	weaponLavaBeam        Weapon = 39
	weaponLavaStream      Weapon = 35
	weaponLavaSpray       Weapon = 36
	weaponSpikeGun        Weapon = 40
	weaponSword           Weapon = 2
	weaponBlinkDagger     Weapon = 4
	weaponSpear           Weapon = 63
	weaponTimeBubble      Weapon = 21
	weaponLaser           Weapon = 14
	weaponIceGun          Weapon = 7
	weaponBlackHole       Weapon = 41
	weaponGlueGun         Weapon = 27
	weaponMinigun         Weapon = 19
	weaponFlameThrower    Weapon = 23
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
