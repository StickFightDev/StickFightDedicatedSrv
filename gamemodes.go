package main

//WeaponSpawnRate holds a spawn rate for weapons
type WeaponSpawnRate struct {
	MinimumSeconds int
	MaximumSeconds int
}

//GameMode holds a Stick Fight game mode
type GameMode interface {
	IsDone() bool                           //Called to check if match processing is finished, if variable must be set to true when GameMode.StartMatch() finishes
	GetLevels() []*Level                    //Returns the allowed levels for this game mode, or nothing if any levels are allowed
	GetWeapons() []Weapon                  //Returns the weapon list that will be in use for this game mode
	GetWeaponSpawnRates() []WeaponSpawnRate //Returns the weapon spawn rates that match the four in-game options (normal, fast, none, slow), with 0/0 for no spawns
	StartMatch(lobby *Lobby)                //Gets called in a goroutine at the start of each match, must allow GameMode.IsDone() to return true if checking lobby.MatchInProgress() to finish running
}
