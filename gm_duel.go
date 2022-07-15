package main

import "time"

//Duel is a competitive duel-style game mode
type Duel struct{}

//IsDone returns true because duels don't need to process after a match
func (gm Duel) IsDone() bool {
	return true
}

//GetLevels returns no levels because duels can use any level
func (gm Duel) GetLevels() []*Level {
	return make([]*Level, 0)
}

//GetWeapons returns the allowed weapons for a duel
func (gm Duel) GetWeapons() []Weapon {
	return []Weapon{
		weaponPistol,
		weaponRevolver,
		weaponDeagle,
		weaponAK47,
		weaponM1,
		weaponSniper,
		weaponSawedOff,
		weaponMilitaryShotgun,
		weaponGrenadeLauncher,
		weaponThruster,
		weaponSnakePistol,
		weaponSnakeShotgun,
		weaponSnakeGrenadeLauncher,
		weaponSpikeBall,
		weaponSpikeGun,
		weaponSword,
		weaponSpear,
		weaponTimeBubble,
		weaponIceGun,
	}
}

//GetWeaponSpawnRates returns the allowed weapon spawn rates for a duel
func (gm Duel) GetWeaponSpawnRates() []WeaponSpawnRate {
	return []WeaponSpawnRate{
		WeaponSpawnRate{
			MinimumSeconds: 3,
			MaximumSeconds: 5,
		},
	}
}

//StartMatch handles what happens when the match starts
func (gm Duel) StartMatch(lobby *Lobby) {
	log.Info("Starting match with gamemode: Duel")

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
