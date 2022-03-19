package main

import "time"

//WeaponSpawnRate holds a spawn rate for weapons
type WeaponSpawnRate struct {
	MinimumSeconds int
	MaximumSeconds int
}

//GameMode holds a Stick Fight game mode
type GameMode interface {
	IsDone() bool                           //Called to check if match processing is finished, if variable must be set to true when GameMode.StartMatch() finishes
	GetLevels() []Level                     //Returns the allowed levels for this game mode, or nothing if any levels are allowed
	GetWeapons() []Weapon                   //Returns the weapon list that will be in use for this game mode
	GetWeaponSpawnRates() []WeaponSpawnRate //Returns the weapon spawn rates that match the four in-game options (normal, fast, none, slow), with 0/0 for no spawns
	StartMatch(lobby *Lobby)                //Gets called in a goroutine at the start of each match, must allow GameMode.IsDone() to return true if checking lobby.MatchInProgress() to finish running
}

//Stock is the default game mode
type Stock struct{}

//IsDone returns true because stock games don't need to process after a match
func (gm Stock) IsDone() bool {
	return true
}

//GetLevels returns no levels because stock games can use any level
func (gm Stock) GetLevels() []Level {
	return make([]Level, 0)
}

//GetWeapons returns the allowed weapons for a stock match
func (gm Stock) GetWeapons() []Weapon {
	return []Weapon{
		weaponPistol, weaponRevolver, weaponDeagle, weaponUzi, weaponGodPistol, weaponAK47,
		weaponM16, weaponM1, weaponSniper, weaponSawedOff, weaponMilitaryShotgun, weaponBouncer,
		weaponGrenadeLauncher, weaponThruster, weaponRPG, weaponSnakePistol, weaponSnakeShotgun, weaponSnakeGrenadeLauncher,
		weaponSnakeLauncher, weaponSnakeMinigun, weaponFlyingSnakeLauncher, weaponSpikeBall, weaponLavaBeam, weaponLavaStream,
		weaponLavaSpray, weaponSpikeGun, weaponSword, weaponBlinkDagger, weaponSpear, weaponTimeBubble,
		weaponLaser, weaponIceGun, weaponBlackHole, weaponGlueGun, weaponMinigun, weaponFlameThrower,
		weaponShield, weaponFan, weaponBall, weaponLavaWhip, weaponMinigunTiny, weaponLaserPlanter,
		weaponHolySword,
	}
}

//GetWeaponSpawnRates returns the allowed weapon spawn rates for a tournament
func (gm Stock) GetWeaponSpawnRates() []WeaponSpawnRate {
	return []WeaponSpawnRate{
		WeaponSpawnRate{
			MinimumSeconds: 5,
			MaximumSeconds: 8,
		},
		WeaponSpawnRate{
			MinimumSeconds: 3,
			MaximumSeconds: 5,
		},
		WeaponSpawnRate{
			MinimumSeconds: 0,
			MaximumSeconds: 0,
		},
		WeaponSpawnRate{
			MinimumSeconds: 8,
			MaximumSeconds: 12,
		},
	}
}

//StartMatch handles what happens when the match starts
func (gm Stock) StartMatch(lobby *Lobby) {
	log.Info("Starting match with gamemode: Stock")

	lastWeaponSpawn := time.Now()
	weaponSpawnWait := randomizer.Intn(gm.GetWeaponSpawnRates()[0].MaximumSeconds-gm.GetWeaponSpawnRates()[0].MinimumSeconds+1) + gm.GetWeaponSpawnRates()[0].MinimumSeconds
	log.Trace("Weapon initial spawn wait: ", weaponSpawnWait)

	for lobby.MatchInProgress() {
		if !lobby.MatchInProgress() {
			break
		}

		if int(time.Now().Sub(lastWeaponSpawn)/time.Second) >= weaponSpawnWait {
			lobby.SpawnWeaponRandom()

			weaponSpawnWait = randomizer.Intn(gm.GetWeaponSpawnRates()[0].MaximumSeconds-gm.GetWeaponSpawnRates()[0].MinimumSeconds+1) + gm.GetWeaponSpawnRates()[0].MinimumSeconds
			log.Trace("Weapon next spawn wait: ", weaponSpawnWait)
			lastWeaponSpawn = time.Now()
		}
	}
}

//Tournament is a competitive tournament-style game mode
type Tournament struct{}

//IsDone returns true because tournaments don't need to process after a match
func (gm Tournament) IsDone() bool {
	return true
}

//GetLevels returns no levels because tournaments can use any level
func (gm Tournament) GetLevels() []Level {
	return make([]Level, 0)
}

//GetWeapons returns the allowed weapons for a tournament
func (gm Tournament) GetWeapons() []Weapon {
	return []Weapon{
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
}

//GetWeaponSpawnRates returns the allowed weapon spawn rates for a tournament
func (gm Tournament) GetWeaponSpawnRates() []WeaponSpawnRate {
	return []WeaponSpawnRate{
		WeaponSpawnRate{
			MinimumSeconds: 3,
			MaximumSeconds: 5,
		},
	}
}

//StartMatch handles what happens when the match starts
func (gm Tournament) StartMatch(lobby *Lobby) {
	log.Info("Starting match with gamemode: Tournament")

	lastWeaponSpawn := time.Now()
	weaponSpawnWait := randomizer.Intn(gm.GetWeaponSpawnRates()[0].MaximumSeconds-gm.GetWeaponSpawnRates()[0].MinimumSeconds+1) + gm.GetWeaponSpawnRates()[0].MinimumSeconds
	log.Trace("Weapon initial spawn wait: ", weaponSpawnWait)

	for lobby.MatchInProgress() {
		if !lobby.MatchInProgress() {
			break
		}

		if int(time.Now().Sub(lastWeaponSpawn)/time.Second) >= weaponSpawnWait {
			lobby.SpawnWeaponRandom()

			weaponSpawnWait = randomizer.Intn(gm.GetWeaponSpawnRates()[0].MaximumSeconds-gm.GetWeaponSpawnRates()[0].MinimumSeconds+1) + gm.GetWeaponSpawnRates()[0].MinimumSeconds
			log.Trace("Weapon next spawn wait: ", weaponSpawnWait)
			lastWeaponSpawn = time.Now()
		}
	}
}

//GunGame is a climb-the-ranks game mode where with each successful kill you progress a weapon for the remainder of the round until your next kill,
//but death by melee will knock you back a weapon
type GunGame struct {
	Done       bool                //If the match is done being processed before the next match can start
	PlayerData []GunGamePlayerData //The data for each player's gun game session
}

//GunGamePlayerData holds the data for a player participating in gun game
type GunGamePlayerData struct {
	Dead        bool //If the player has been pronounced dead
	WeaponIndex int  //The current index into the weapon list
}

//IsDone returns if a match is done being processed
func (gm GunGame) IsDone() bool {
	return gm.Done
}

//GetLevels returns no levels because gun game can use any level
func (gm GunGame) GetLevels() []Level {
	return make([]Level, 0)
}

//GetWeapons returns the weapons that will be used by the gun game in order of progression
func (gm GunGame) GetWeapons() []Weapon {
	return []Weapon{
		weaponPistol, weaponRevolver, weaponSpikeGun, weaponDeagle,
		weaponUzi, weaponIceGun, weaponLavaSpray,
		weaponM1, weaponSniper, weaponBouncer, weaponLavaBeam,
		weaponAK47, weaponM16, weaponMinigun, weaponLaser,
		weaponMilitaryShotgun, weaponSawedOff,
		weaponGlueGun, weaponTimeBubble, weaponPumpkinShooter, weaponFlameThrower, weaponLavaStream,
		weaponThruster, weaponGrenadeLauncher, weaponSpikeBall, weaponRPG, weaponGodPistol,
		weaponSpear, weaponSword, weaponLightsaber, weaponHolySword, weaponBlinkDagger,
		weaponBlackHole,
	}
}

//GetWeaponSpawnRates returns the allowed weapon spawn rates for a tournament
func (gm GunGame) GetWeaponSpawnRates() []WeaponSpawnRate {
	return []WeaponSpawnRate{
		WeaponSpawnRate{
			MinimumSeconds: 0,
			MaximumSeconds: 0,
		},
	}
}

//StartMatch handles what happens when the match starts
func (gm GunGame) StartMatch(lobby *Lobby) {
	log.Info("Starting match with gamemode: Gun Game")

	gm.Done = false

	//Prepare the player data for this match
	for playerIndex := 0; playerIndex < len(gm.PlayerData); playerIndex++ {
		lobby.UpdateWeapon(playerIndex, gm.GetWeapons()[gm.PlayerData[playerIndex].WeaponIndex])
		gm.PlayerData[playerIndex].Dead = false
		log.Trace("-- [Gun Game] Player ", playerIndex, " is no longer processed!")
	}

	//Loop until the match is over
	for lobby.MatchInProgress() {
		if lobby == nil || !lobby.IsRunning() || !lobby.MatchInProgress() {
			break
		}
		time.Sleep(time.Millisecond * 10)

		players := lobby.GetPlayers()
		if len(players) > 0 {
			if len(players) > len(gm.PlayerData) {
				newPlayers := len(players) - len(gm.PlayerData)
				for i := 0; i < newPlayers; i++ {
					gm.PlayerData = append(gm.PlayerData, GunGamePlayerData{})
				}
				log.Trace("-- [Gun Game] Added ", newPlayers, " players")
			}

			for playerIndex := 0; playerIndex < len(players); playerIndex++ {
				if players[playerIndex] != nil {
					lastAttackerIndex := players[playerIndex].LastAttackerIndex
					lastAttackerWeapon := players[lastAttackerIndex].Weapon.Weapon
					lastAttackerWeaponIndex := gm.PlayerData[lastAttackerIndex].WeaponIndex
					playerWeapon := players[playerIndex].Weapon.Weapon
					playerWeaponIndex := gm.PlayerData[playerIndex].WeaponIndex

					if playerWeapon != weaponEmpty && playerWeapon != gm.GetWeapons()[playerWeaponIndex] {
						lobby.UpdateWeapon(playerIndex, weaponEmpty)
					}

					if !gm.PlayerData[playerIndex].Dead && players[playerIndex].Health <= 0 {
						log.Trace("-- [Gun Game] Player ", playerIndex, " died from player ", lastAttackerIndex, " and needs to be processed!")
						gm.PlayerData[playerIndex].Dead = true

						if lastAttackerIndex != playerIndex {
							if lastAttackerWeapon == gm.GetWeapons()[lastAttackerWeaponIndex] {
								if lastAttackerWeaponIndex != len(gm.GetWeapons()) {
									gm.PlayerData[lastAttackerIndex].WeaponIndex++
									log.Trace("-- [Gun Game] Increased player ", lastAttackerIndex, " to ", lastAttackerWeaponIndex+1)
									lobby.UpdateWeapon(lastAttackerIndex, gm.GetWeapons()[lastAttackerWeaponIndex+1])
								} else {
									log.Trace("-- [Gun Game] Player ", lastAttackerIndex, " is the gun game winner!")

									for playerIndex := 0; playerIndex < len(gm.PlayerData); playerIndex++ {
										gm.PlayerData[playerIndex].Dead = false
										gm.PlayerData[playerIndex].WeaponIndex = 0
									}

									lobby.PlayerSaid(lastAttackerIndex, "I'm the Gun Game winner!")
								}
							} else {
								if playerWeaponIndex != 0 {
									gm.PlayerData[playerIndex].WeaponIndex--
									log.Trace("-- [Gun Game] Decreased player ", playerIndex, " to ", gm.GetWeapons()[playerWeaponIndex-1])
								}
							}
						}
					}

				}
			}
		}
	}

	if len(lobby.GetPlayers()) != len(gm.PlayerData) {
		log.Trace("-- [Gun Game] Player count changed, resetting!")
		gm.PlayerData = make([]GunGamePlayerData, lobby.GetPlayerCount(false))
		lobby.PlayerSaid(0, "Player count changed,\nreset Gun Game!")
	}

	gm.Done = true
}
