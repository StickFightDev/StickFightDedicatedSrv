package main

var (
	validWeapons = []Weapon{
		weaponPistol, weaponRevolver, weaponDeagle, weaponUzi, weaponGodPistol, weaponAK47,
		weaponM16, weaponM1, weaponSniper, weaponSawedOff, weaponMilitaryShotgun, weaponBouncer,
		weaponGrenadeLauncher, weaponThruster, weaponRPG, weaponSnakePistol, weaponSnakeShotgun, weaponSnakeGrenade,
		weaponSnakeLauncher, weaponSnakeMinigun, weaponFlyingSnake, weaponSpikeBall, weaponLavaBeam, weaponLavaStream,
		weaponLavaSpray, weaponSpikeGun, weaponSword, weaponBlinkDagger, weaponSpear, weaponTimeBubble,
		weaponLaser, weaponIceGun, weaponBlackHole, weaponGlueGun, weaponMinigun, weaponFlameThrower,
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
