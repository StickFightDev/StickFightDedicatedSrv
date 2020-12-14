package main

import "net"

type player struct {
	Addr    *net.UDPAddr
	SteamID uint64
	Stats   playerStats
	Status  playerStatus

	UpdateChannel int
	EventChannel  int
}

type playerStats struct {
	Wins            int32
	Kills           int32
	Deaths          int32
	Suicides        int32
	Falls           int32
	CrownSteals     int32
	BulletsHit      int32
	BulletsMissed   int32
	BulletsShot     int32
	Blocks          int32
	PunchesLanded   int32
	WeaponsPickedUp int32
	WeaponsThrown   int32
}

type playerStatus struct {
	IsRed             bool
	LastTick          uint64
	PingInMs          float64
	ControlledLocally bool
	IsSocket          bool
	Ready             bool
	Position          networkPosition
	Weapon            networkWeapon
	//PlayerObject *gameObject

	//Additional statuses
	Spawned bool    //Whether or not the player is currently spawned, becomes false upon death
	Health  float32 //Player health
	Dead    bool    //Whether or not the player is dead
	Moved   bool    //If the player has sent a playerUpdate packet
}

type networkPosition struct {
	Position, Rotation vector2
	YValue             int
	MovementType       movementType
}

type networkWeapon struct {
	FightState  fightState
	Projectiles []projectile
	WeaponType  weaponType
}

type movementType byte

type fightState byte

type weaponType byte

type projectile struct {
	Shoot         vector3
	ShootPosition vector3
	SyncIndex     uint16
}

type damageType byte

const (
	damageTypePunch damageType = iota
	damageTypeLocalDamage
	damageTypeOther
)

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
