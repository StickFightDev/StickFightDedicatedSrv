package main

import "fmt"

var (
	validWeapons = []Weapon{
		weaponPistol, weaponRevolver, weaponDeagle, weaponUzi, weaponGodPistol, weaponAK47,
		weaponM16, weaponM1, weaponSniper, weaponSawedOff, weaponMilitaryShotgun, weaponBouncer,
		weaponGrenadeLauncher, weaponThruster, weaponRPG, weaponSnakePistol, weaponSnakeShotgun, weaponSnakeGrenadeLauncher,
		weaponSnakeLauncher, weaponSnakeMinigun, weaponFlyingSnakeLauncher, weaponSpikeBall, weaponLavaBeam, weaponLavaStream,
		weaponLavaSpray, weaponSpikeGun, weaponSword, weaponBlinkDagger, weaponSpear, weaponTimeBubble,
		weaponLaser, weaponIceGun, weaponBlackHole, weaponGlueGun, weaponMinigun, weaponFlameThrower,
		weaponShield, weaponFan, weaponBall, weaponLavaWhip, weaponMinigunTiny, weaponLaserPlanter,
		weaponHolySword,
	}
	tourneyWeapons = []Weapon{
		weaponPistol,
		weaponRevolver,
		weaponDeagle,
		weaponM1,
		weaponSniper,
		weaponMilitaryShotgun,
		weaponGrenadeLauncher,
		weaponThruster,
		weaponSnakePistol,
		weaponSnakeLauncher,
		weaponSword,
		weaponSpear,
		weaponIceGun,
	}
)

//Weapon is the weapon ID of a weapon
type Weapon int

func (weapon Weapon) String() string {
	switch weapon {
	case weaponEmpty:
		return "Empty"
	case weaponPistol:
		return "Pistol"
	case weaponRevolver:
		return "Revolver"
	case weaponDeagle:
		return "Deagle"
	case weaponUzi:
		return "Uzi"
	case weaponGodPistol:
		return "God Pistol"
	case weaponAK47:
		return "AK-47"
	case weaponM16:
		return "M16"
	case weaponM1:
		return "M1"
	case weaponSniper:
		return "Sniper"
	case weaponSawedOff:
		return "Sawed Off"
	case weaponMilitaryShotgun:
		return "Military Shotgun"
	case weaponBouncer:
		return "Bouncer"
	case weaponGrenadeLauncher:
		return "Grenade Launcher"
	case weaponThruster:
		return "Thruster"
	case weaponRPG:
		return "RPG"
	case weaponSnakePistol:
		return "Snake Pistol"
	case weaponSnakeShotgun:
		return "Snake Shotgun"
	case weaponSnakeGrenadeLauncher:
		return "Snake Grenade Launcher"
	case weaponSnakeLauncher:
		return "Snake Launcher"
	case weaponSnakeMinigun:
		return "Snake Minigun"
	case weaponFlyingSnakeLauncher:
		return "Flying Snake Launcher"
	case weaponSpikeBall:
		return "Spike Ball"
	case weaponLavaBeam:
		return "Lava Beam"
	case weaponLavaStream:
		return "Lava Stream"
	case weaponLavaSpray:
		return "Lava Spray"
	case weaponSpikeGun:
		return "Spike Gun"
	case weaponSword:
		return "Sword"
	case weaponBlinkDagger:
		return "Blink Dagger"
	case weaponSpear:
		return "Spear"
	case weaponTimeBubble:
		return "Time Bubble"
	case weaponLaser:
		return "Laser"
	case weaponIceGun:
		return "Ice Gun"
	case weaponBlackHole:
		return "Black Hole"
	case weaponGlueGun:
		return "Glue Gun"
	case weaponMinigun:
		return "Minigun"
	case weaponFlameThrower:
		return "Flame Thrower"
	case weaponShield:
		return "Shield"
	case weaponFan:
		return "Fan"
	case weaponBall:
		return "Ball"
	case weaponMinigunTiny:
		return "Minigun Tiny"
	case weaponLaserPlanter:
		return "Laser Planter"
	case weaponHolySword:
		return "Holy Sword"
	case weaponGodMinigun:
		return "God Minigun"
	case weaponLavaWhip:
		return "Lava Whip"
	case weaponPumpkinShooter:
		return "Pumpkin Shooter"
	case weaponLightsaber:
		return "Lightsaber"
	}

	return fmt.Sprintf("unknown%d", weapon)
}

const (
	//Weapons officially supported by the game
	weaponEmpty                Weapon = 0
	weaponPistol               Weapon = 1
	weaponAK47                 Weapon = 2
	weaponSword                Weapon = 3
	weaponGrenadeLauncher      Weapon = 4
	weaponBlinkDagger          Weapon = 5
	weaponSniper               Weapon = 6
	weaponRevolver             Weapon = 7
	weaponIceGun               Weapon = 8
	weaponMilitaryShotgun      Weapon = 11
	weaponThruster             Weapon = 13
	weaponLaser                Weapon = 15
	weaponUzi                  Weapon = 17
	weaponMinigun              Weapon = 20
	weaponBouncer              Weapon = 21
	weaponTimeBubble           Weapon = 22
	weaponRPG                  Weapon = 23
	weaponFlameThrower         Weapon = 24
	weaponSnakePistol          Weapon = 25
	weaponSnakeGrenadeLauncher Weapon = 26
	weaponSnakeLauncher        Weapon = 27
	weaponGlueGun              Weapon = 28
	weaponGodPistol            Weapon = 32
	weaponM1                   Weapon = 33
	weaponSnakeMinigun         Weapon = 34
	weaponLavaStream           Weapon = 36
	weaponLavaSpray            Weapon = 37
	weaponSnakeShotgun         Weapon = 38
	weaponSpikeBall            Weapon = 39
	weaponLavaBeam             Weapon = 40
	weaponSpikeGun             Weapon = 41
	weaponBlackHole            Weapon = 42
	weaponM16                  Weapon = 61
	weaponDeagle               Weapon = 62
	weaponSawedOff             Weapon = 63
	weaponSpear                Weapon = 64
	weaponFlyingSnakeLauncher  Weapon = 65

	//Weapons not exposed by the game
	weaponShield                Weapon = 9
	weaponFan                   Weapon = 10
	weaponBall                  Weapon = 12
	weaponBowAndArrow           Weapon = 14
	weaponLightsaber            Weapon = 16
	weaponMinigunMediumSilenced Weapon = 18
	weaponMinigunTiny           Weapon = 19
	weaponLaserPlanter          Weapon = 29
	weaponHolySword             Weapon = 30
	weaponGodMinigun            Weapon = 31
	weaponLavaWhip              Weapon = 35
	weaponBoss1TrinityTinyBall  Weapon = 43
	weaponBoss2Spike            Weapon = 44
	weaponBoss3Trinity          Weapon = 45
	weaponAK47ExtendedMag       Weapon = 46
	weaponBoss2Spear            Weapon = 47
	weaponBoss2TinyBall         Weapon = 48
	weaponBoss1TinyBall         Weapon = 49
	weaponBoss3Shield           Weapon = 50
	weaponBoss3TinyBall         Weapon = 51
	weaponRevolverUnlimited     Weapon = 52
	weaponBoss3Shield2          Weapon = 53
	weaponPistolExtendedMag     Weapon = 54
	weaponBoss4FloorSpreader    Weapon = 55
	weaponBoss4CircleOfLife     Weapon = 56
	weaponBoss4TinyBall         Weapon = 57
	weaponSniperExtendedMag     Weapon = 58
	weaponBoss1Snowflakes       Weapon = 59
	weaponPumpkinShooter        Weapon = 60
)

//NetworkWeapon holds a player's current weapon according to the network
type NetworkWeapon struct {
	FightState  FightState
	Weapon      Weapon
	Projectiles []Projectile
}

//FightState is the fighting state of a player
type FightState byte

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
