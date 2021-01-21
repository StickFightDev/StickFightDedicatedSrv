package main

//Player holds a Stick Fight player
type Player struct {
	Client *Client //The client that's hosting this player

	//Player session tracking
	Index    int             //The index of the player array where Stick Fight clients expect to find this player
	Stats    PlayerStats     //The player's statistics for the match session so far
	Health   float32         //The current health of the player
	Ready    bool            //If the player is ready for the next match
	Spawned  bool            //If the server has spawned the player already
	Position NetworkPosition //The current position of the player
	Weapon   NetworkWeapon   //The current weapon of the player
}

//GetChannelUpdate returns the channel that update packets are expected on
func (player *Player) GetChannelUpdate() int {
	return player.Index*2 + 2
}

//GetChannelEvent returns the channel that event packets are expected on
func (player *Player) GetChannelEvent() int {
	return player.GetChannelUpdate() + 1
}

//IsDead returns true if the player's health is equal to or below 0
func (player *Player) IsDead() bool {
	return player.Health <= 0
}

//IsReady returns whether or not the player is ready
func (player *Player) IsReady() bool {
	return player.Ready
}

//SetReady sets the player's ready status
func (player *Player) SetReady(ready bool) {
	player.Ready = ready
}

//SetPosition sets a player's network position
func (player *Player) SetPosition(posX, posY, rotX, rotY float32, yValue int, movementType MovementType) {
	player.Position = NetworkPosition{
		Position:     Vector2{posX, posY},
		Rotation:     Vector2{rotX, rotY},
		YValue:       yValue,
		MovementType: movementType,
	}
}

//SetWeapon sets a player's network weapon
func (player *Player) SetWeapon(fightState FightState, weaponType WeaponType, projectiles []Projectile) {
	player.Weapon = NetworkWeapon{
		FightState:  fightState,
		WeaponType:  weaponType,
		Projectiles: projectiles,
	}
}

//PlayerStats holds the statistics of a player's match session so far
type PlayerStats struct {
	Wins, Kills, Deaths, Suicides, Falls   int32 //The death of a player
	CrownSteals                            int32 //How many times you've stolen the crown from another player
	BulletsHit, BulletsMissed, BulletsShot int32 //Bullets do a lot of damage
	Blocks, PunchesLanded                  int32 //Hand to hand combat at its finest
	WeaponsPickedUp, WeaponsThrown         int32 //Why shoot a gun when you can throw it?
}

//NetworkPosition holds a player's current position according to the network
type NetworkPosition struct {
	Position, Rotation Vector2
	YValue             int
	MovementType       MovementType
}

//MovementType is the type of player movement
type MovementType byte
