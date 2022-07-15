package main

import "time"

//Stock is the default game mode
type Stock struct{}

//IsDone returns true because stock games don't need to process after a match
func (gm Stock) IsDone() bool {
	return true
}

//GetLevels returns no levels because stock games can use any level
func (gm Stock) GetLevels() []*Level {
	return make([]*Level, 0)
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
