package main

import "time"

//Tournament is a competitive tournament-style game mode
type Tournament struct{}

//IsDone returns true because tournaments don't need to process after a match
func (gm Tournament) IsDone() bool {
	return true
}

//GetLevels returns no levels because tournaments can use any level
func (gm Tournament) GetLevels() []*Level {
	return make([]*Level, 0)
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
