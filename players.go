package main

import "net"

type player struct {
	Addr    *net.UDPAddr
	SteamID uint64
	Stats   playerStats
	Status  playerStatus
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
	Position          vector3
	Rotation          vector3
	//PlayerObject *gameObject

	//Additional statuses
	HasSpawned bool    //Whether or not the player has ever spawned before
	Spawned    bool    //Whether or not the player is currently spawned, becomes false upon death
	Health     float32 //Player health
	Dead       bool    //Whether or not the player is dead
}
