package main

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

//Lobby holds a Stick Fight lobby
type Lobby struct {
	//We don't want race conditions with such a latent game
	sync.Mutex

	Server *Server //The server that's hosting this lobby

	//Lobby settings
	MaxPlayers         int        //The maximum amount of players allowed at any one time
	Health             byte       //The starting health of all players (enum 100, 200, 300, 1, 25, 50, 75)
	Regen              byte       //If health regeneration should be enabled
	WeaponSpawnRateMin int        //The minimum amount of seconds to wait before spawning a new weapon, 0 to disable weapon spawning
	WeaponSpawnRateMax int        //The maximum amount of seconds to wait before spawning a new weapon, 0 to disable weapon spawning
	Public             bool       //If false, requires an invitation from the lobby owner to join
	TourneyRules       bool       //If enabled, tourney rules will be in effect and override stock game rules
	Invited            []CSteamID //A list of invited SteamIDs
	RandomMaps         bool       //If the map rotation should be randomized or in order

	//Session tracker
	Running                       bool      //If the lobby is currently running
	LobbyOwner                    CSteamID  //The current owner of the lobby
	LobbyCreationTime             time.Time //The time of the lobby's creation
	LastTimestamp                 uint32    //The timestamp of the last packet that was accepted by the lobby as present time
	CurrentLevel                  *Level    //The currently-loaded level
	InFight                       bool      //If the match is in progress
	FightStartTime                time.Time //The match's start time
	CompletedLevelsSinceLastStats int       //The amount of matches played so far since the last time the stats map was used
	LastAppliedScale              float32   //The scale to use when managing coordinates on the map
	LastSpawnedWeaponOnLeftSide   bool      //If the last weapon was spawned on the left side or not
	LastSpawnedWeaponTime         time.Time //The last time a weapon was spawned
	CheckingWinner                bool      //Stops multiple CheckWinner calls from happening concurrently

	Clients []*Client //The Stick Fight clients currently connected to this lobby
	Levels  []*Level  //The Stick Fight maps to
}

//NewLobby retuns a new lobby
func NewLobby(srv *Server) (*Lobby, error) {
	if len(srv.Lobbies) >= maxLobbies {
		return nil, errors.New("too many lobbies")
	}

	lobby := &Lobby{
		Running:            true,                                             //Mark this lobby as running
		LobbyCreationTime:  time.Now(),                                       //Set the lobby's creation time to now
		Server:             srv,                                              //A pointer to this lobby's host server
		MaxPlayers:         4,                                                //Default to a max of 4 players, as expected by the stock game
		WeaponSpawnRateMin: 5,                                                //Default to one weapon at least for every 5 seconds
		WeaponSpawnRateMax: 8,                                                //Default to one weapon at max for every 8 seconds
		CurrentLevel:       lobbyLevels[randomizer.Intn(len(lobbyLevels)-1)], //Default to a random lobby map
		LastAppliedScale:   1.0,                                              //The last applied map scaling, used to scale objects and other positions on the map
		Clients:            make([]*Client, 0),                               //Initialize the clients slice
		Levels:             defaultLevels,                                    //Default to the default levels list
	}

	return lobby, nil
}

//IsRunning returns true if the lobby is currently running
func (lobby *Lobby) IsRunning() bool {
	lobby.Lock()
	defer lobby.Unlock()
	return lobby.Running
}

//Close closes the lobby
func (lobby *Lobby) Close() {
	if !lobby.IsRunning() {
		return
	}

	log.Info("Closing lobby!")

	for _, client := range lobby.Clients {
		client.Close()
	}
	lobby.MaxPlayers = 0
	lobby.CurrentLevel = nil
	lobby.FightStartTime = time.Time{}
	lobby.CompletedLevelsSinceLastStats = 0
	lobby.Clients = nil
	lobby.Levels = nil
	lobby.Running = false
}

//BroadcastPacket broadcasts a packet to every client in the lobby, except ignoreAddr if specified
func (lobby *Lobby) BroadcastPacket(packet *Packet, ignoreAddr *net.UDPAddr) {
	if !lobby.IsRunning() {
		return
	}

	for clientIndex := 0; clientIndex < len(lobby.Clients); clientIndex++ {
		if lobby.Clients[clientIndex] != nil {
			if ignoreAddr != nil && ignoreAddr.String() == lobby.Clients[clientIndex].Addr.String() {
				continue //Ignore this address
			}
			lobby.Server.SendPacket(packet, lobby.Clients[clientIndex].Addr)
		}
	}

	if packet.ShouldLog() {
		log.Trace("Broadcasted packet: ", packet)
	}
}

//Handle handles a packet in the lobby
func (lobby *Lobby) Handle(packet *Packet) {
	if !lobby.IsRunning() {
		return
	}

	if packet.ShouldCheckTime() {
		//Check the timestamp!
		if packet.Timestamp < lobby.LastTimestamp {
			log.Warn("Packet from ", packet.Src, " too old: ", packet)
			return
		}
		/*
			if packet.Timestamp > uint32(time.Now().Unix()) {
				log.Warn("Packet from ", packet.Src, " too new: ", packet)
				return
			}
		*/
		lobby.LastTimestamp = packet.Timestamp
	}

	switch packet.Type {
	case packetTypePing:
		if packet.SteamID.ID != 0 {
			_, sourceClient := lobby.GetClientByAddr(packet.Src)
			targetClient := lobby.GetClientBySteamID(packet.SteamID)
			if sourceClient != nil && targetClient != nil {
				packet.SteamID = sourceClient.SteamID
				lobby.Server.SendPacket(packet, targetClient.Addr)
			}
		} else {
			packet.Type = packetTypePingResponse
			lobby.Server.SendPacket(packet, packet.Src)
		}

	case packetTypePingResponse:
		if packet.SteamID.ID != 0 {
			_, sourceClient := lobby.GetClientByAddr(packet.Src)
			targetClient := lobby.GetClientBySteamID(packet.SteamID)
			if sourceClient != nil && targetClient != nil {
				packet.SteamID = sourceClient.SteamID
				lobby.Server.SendPacket(packet, targetClient.Addr)
			}
		}

	case packetTypeClientRequestingToSpawn:
		playerIndex := int(packet.ReadByteNext())
		player := lobby.GetPlayerByIndex(playerIndex)
		if player == nil {
			log.Error("Unable to spawn invalid player ", playerIndex)
			return
		}
		if player.Client.Addr.String() != packet.Src.String() {
			log.Error("Client ", packet.Src, " is trying to spawn player ", playerIndex, " from client ", player.Client.Addr)
			return
		}

		lobby.SpawnPlayer(playerIndex, packet.ReadF32LENext(1)[0], packet.ReadF32LENext(1)[0], packet.ReadF32LENext(1)[0], packet.ReadF32LENext(1)[0])

	case packetTypeLobbyType:
		_, client := lobby.GetClientByAddr(packet.Src)
		playerIndex := client.Players[0].Index

		if lobby.IsOwner(client.SteamID) {
			flag := int(packet.ReadByteNext())
			switch flag {
			case 1: //Friends only
				lobby.Public = false
				lobby.PlayerSaid(playerIndex, "Set lobby to private!")
			case 2: //Public
				lobby.Public = true
				lobby.PlayerSaid(playerIndex, "Set lobby to public!")
			default:
				lobby.PlayerSaid(playerIndex, "Unhandled lobby type %d!", flag)
			}
		} else {
			lobby.PlayerSaid(playerIndex, "No permissions!")
		}

	case packetTypeClientReadyUp:
		lobby.ReadyUp(packet)

	case packetTypeStartMatch:
		lobby.StartMatch()

	case packetTypeKickPlayer, packetTypeClientLeft:
		_, client := lobby.GetClientByAddr(packet.Src)
		if client != nil {
			lobby.KickClientBySteamID(client.SteamID.ID)
		}

	case packetTypePlayerTalked:
		lobby.PlayerTalked(packet)

	case packetTypePlayerUpdate:
		lobby.PlayerUpdate(packet)

	case packetTypePlayerTookDamage:
		lobby.PlayerTookDamage(packet)

	case packetTypePlayerFallOut:
		lobby.PlayerFallOut(packet)

	case packetTypePlayerForceAdded:
		lobby.BroadcastPacket(packet, packet.Src)

	case packetTypePlayerForceAddedAndBlock:
		lobby.BroadcastPacket(packet, packet.Src)

	case packetTypePlayerLavaForceAdded:
		lobby.BroadcastPacket(packet, packet.Src)

	case packetTypeClientRequestingWeaponDrop:
		nextWeaponSpawnID := lobby.GetNextWeaponSpawnID(false)
		nextObjectSpawnID := lobby.GetNextObjectSpawnID(false)

		packet.Type = packetTypeWeaponDropped
		packet.Grow(4)
		packet.WriteU16LENext([]uint16{nextWeaponSpawnID, nextObjectSpawnID})

		log.Info("Weapon ", int(packet.ReadByte(0x0)), " was dropped!")
		lobby.BroadcastPacket(packet, nil)

	case packetTypeClientRequestingWeaponPickUp:
		playerIndex := int(packet.ReadByteNext())
		weaponSpawnID := packet.ReadU16LENext(1)[0]

		if weapon, ok := lobby.CurrentLevel.SpawnedWeapons[weaponSpawnID]; ok && weapon != nil {
			packet.Type = packetTypeWeaponWasPickedUp

			log.Info("Player ", playerIndex, " picked up weapon ", weaponSpawnID, "!")
			lobby.BroadcastPacket(packet, nil)
		} else {
			log.Error("Player ", playerIndex, " tried to pick up invalid weapon ", weaponSpawnID, "!")
		}

	case packetTypeClientRequestingWeaponThrow:
		nextWeaponSpawnID := lobby.GetNextWeaponSpawnID(false)
		nextObjectSpawnID := lobby.GetNextObjectSpawnID(false)

		packet.Type = packetTypeWeaponThrown
		packet.Grow(4)
		packet.WriteU16LE(packet.ByteCapacity()-4, []uint16{nextWeaponSpawnID, nextObjectSpawnID})

		log.Info("Weapon ", int(packet.ReadByte(0x0)), " was thrown!")
		lobby.BroadcastPacket(packet, nil)

	default:
		log.Error(fmt.Sprintf("Unhandled packet from %s: %s", packet.Src, packet))
	}
}

//GetMaxHealth returns the maximum and starting health of a player
func (lobby *Lobby) GetMaxHealth() float32 {
	switch lobby.Health {
	case 0:
		return 100
	case 1:
		return 200
	case 2:
		return 300
	case 3:
		return 1
	case 4:
		return 25
	case 5:
		return 50
	case 6:
		return 75
	}

	return 0
}

//GetNextWeaponSpawnID returns the next available weaponSpawnID
func (lobby *Lobby) GetNextWeaponSpawnID(beginFromEnd bool) uint16 {
	if !lobby.IsRunning() {
		return 0
	}

	if lobby.CurrentLevel.SpawnedWeapons == nil {
		lobby.CurrentLevel.SpawnedWeapons = make(map[uint16]*SyncableWeapon)
	}

	weaponSpawnID := uint16(65534)
	if beginFromEnd {
		weaponSpawnID = uint16(len(lobby.CurrentLevel.SpawnedWeapons))
	}

	for {
		//log.Trace("Trying weapon spawn ID ", weaponSpawnID)
		if _, ok := lobby.CurrentLevel.SpawnedWeapons[weaponSpawnID]; !ok {
			break
		}

		if beginFromEnd {
			weaponSpawnID--
		} else {
			weaponSpawnID++
		}
	}

	lobby.CurrentLevel.SpawnedWeapons[weaponSpawnID] = &SyncableWeapon{}
	return weaponSpawnID
}

//GetNextObjectSpawnID returns the next available objectSpawnID
func (lobby *Lobby) GetNextObjectSpawnID(beginFromEnd bool) uint16 {
	if !lobby.IsRunning() {
		return 0
	}

	if lobby.CurrentLevel.SpawnedObjects == nil {
		lobby.CurrentLevel.SpawnedObjects = make(map[uint16]*SyncableObject)
	}

	objectSpawnID := uint16(65534)
	if beginFromEnd {
		objectSpawnID = uint16(len(lobby.CurrentLevel.SpawnedObjects))
	}

	for {
		//log.Trace("Trying object spawn ID ", objectSpawnID)
		if _, ok := lobby.CurrentLevel.SpawnedObjects[objectSpawnID]; !ok {
			break
		}

		if beginFromEnd {
			objectSpawnID--
		} else {
			objectSpawnID++
		}
	}

	lobby.CurrentLevel.SpawnedObjects[objectSpawnID] = &SyncableObject{}
	return objectSpawnID
}

//GetPlayerCount returns how many players are in this lobby
func (lobby *Lobby) GetPlayerCount(excludeSelf bool) int {
	playerCount := 0
	if len(lobby.Clients) > 0 {
		for _, client := range lobby.Clients {
			playerCount += client.GetPlayerCount()
		}
	}
	if excludeSelf && playerCount > 0 {
		playerCount--
	}
	return playerCount
}

//GetPlayersTooMany returns true if the current player count plus the playersToAdd count exceeds the lobby's maximum player setting
func (lobby *Lobby) GetPlayersTooMany(playersToAdd int, excludeSelf bool) bool {
	if !lobby.IsRunning() {
		return true
	}

	return lobby.GetPlayerCount(excludeSelf)+playersToAdd > lobby.MaxPlayers
}

//GetPlayers returns the current player list in order of playerIndex
func (lobby *Lobby) GetPlayers() []*Player {
	if lobby.Clients == nil || len(lobby.Clients) == 0 {
		return make([]*Player, 0)
	}

	players := make(map[int]*Player)
	for clientIndex := 0; clientIndex < len(lobby.Clients); clientIndex++ {
		for playerIndex := 0; playerIndex < lobby.Clients[clientIndex].GetPlayerCount(); playerIndex++ {
			players[lobby.Clients[clientIndex].Players[playerIndex].Index] = lobby.Clients[clientIndex].Players[playerIndex]
		}
	}

	playerList := make([]*Player, lobby.MaxPlayers)
	for playerIndex := 0; playerIndex < lobby.MaxPlayers; playerIndex++ {
		if player, ok := players[playerIndex]; ok {
			playerList[playerIndex] = player
		} else {
			playerList[playerIndex] = nil
		}
	}

	return playerList
}

//GetPlayerByIndex returns the player with a matching index
func (lobby *Lobby) GetPlayerByIndex(index int) *Player {
	if lobby.Clients == nil || len(lobby.Clients) == 0 {
		return nil
	}

	players := lobby.GetPlayers()
	if index >= len(players) {
		return nil
	}

	return players[index]
}

//GetNextPlayerIndex returns the next available playerIndex
func (lobby *Lobby) GetNextPlayerIndex() int {
	if !lobby.IsRunning() {
		return -1
	}

	if lobby.GetPlayersTooMany(1, true) {
		return -1
	}
	if lobby.Clients == nil || len(lobby.Clients) == 0 {
		return 0
	}

	usedIndexes := make(map[int]bool)
	for clientIndex := 0; clientIndex < len(lobby.Clients); clientIndex++ {
		for playerIndex := 0; playerIndex < lobby.Clients[clientIndex].GetPlayerCount(); playerIndex++ {
			if lobby.Clients[clientIndex].Players[playerIndex].Index > -1 {
				usedIndexes[lobby.Clients[clientIndex].Players[playerIndex].Index] = true
			}
		}
	}

	nextPlayerIndex := 0
	for {
		if lobby.GetPlayersTooMany(1, true) {
			return -1
		}
		if isUsed, ok := usedIndexes[nextPlayerIndex]; ok && isUsed {
			nextPlayerIndex++
			continue
		}
		break
	}

	log.Trace("Next player index: ", nextPlayerIndex)
	return nextPlayerIndex
}

//GetClientByAddr returns the client with a matching address
func (lobby *Lobby) GetClientByAddr(addr *net.UDPAddr) (int, *Client) {
	if lobby.Clients == nil || len(lobby.Clients) == 0 {
		return -1, nil
	}

	for clientIndex, client := range lobby.Clients {
		if client.Addr.String() == addr.String() {
			return clientIndex, client
		}
	}
	return -1, nil
}

//GetClientBySteamID returns the client with a matching SteamID
func (lobby *Lobby) GetClientBySteamID(steamID CSteamID) *Client {
	if lobby.Clients == nil || len(lobby.Clients) == 0 {
		return nil
	}

	for _, client := range lobby.Clients {
		if client.SteamID.CompareCSteamID(steamID) {
			return client
		}
	}
	return nil
}

//GetIndexesByPlayerIndex returns the index of the client list and the index of the client's player list that matches the specified player index
func (lobby *Lobby) GetIndexesByPlayerIndex(index int) (int, int) {
	if lobby.Clients == nil || len(lobby.Clients) == 0 {
		return -1, -1
	}

	for clientIndex := 0; clientIndex < len(lobby.Clients); clientIndex++ {
		for playerIndex := 0; playerIndex < lobby.Clients[clientIndex].GetPlayerCount(); playerIndex++ {
			if lobby.Clients[clientIndex].Players[playerIndex].Index == index {
				return clientIndex, playerIndex
			}
		}
	}

	return -1, -1
}

//KickClientBySteamID kicks all clients from the lobby that have a matching SteamID
func (lobby *Lobby) KickClientBySteamID(steamID uint64) {
	if !lobby.IsRunning() {
		return
	}

	if lobby.Clients == nil || len(lobby.Clients) == 0 {
		return
	}

	for clientIndex := 0; clientIndex < len(lobby.Clients); clientIndex++ {
		if lobby.Clients[clientIndex].SteamID.CompareSteamID(steamID) {
			lobby.ClientRemoveByClientIndex(clientIndex)
		}
	}
}

//IsInvited returns true if the specified SteamID was invited to the server
func (lobby *Lobby) IsInvited(steamID uint64) bool {
	if !lobby.IsRunning() {
		return false
	}

	if lobby.Public {
		return true
	}

	if len(lobby.GetPlayers()) == 0 {
		return true
	}

	for _, invited := range lobby.Invited {
		if invited.CompareSteamID(steamID) {
			return true
		}
	}
	return false
}

//IsOwner returns true if the specified SteamID is the owner of the lobby
func (lobby *Lobby) IsOwner(steamID CSteamID) bool {
	if !lobby.IsRunning() {
		return false
	}

	return lobby.LobbyOwner.CompareCSteamID(steamID)
}

//ClientInit initializes a client and returns an error if it fails
func (lobby *Lobby) ClientInit(packet *Packet) error {
	if !lobby.IsRunning() {
		return errors.New("lobby not running")
	}

	packet.SeekByte(0, false) //Seek to the start of the packet data

	steamID := packet.ReadU64LENext(1)[0] //Read in the SteamID
	lobby.KickClientBySteamID(steamID)    //Remove this player from the lobby if they currently exist in it

	//Make sure this player is allowed in the lobby
	if !lobby.IsInvited(steamID) {
		return fmt.Errorf("not invited to this lobby")
	}

	clientPlayerCount := int(packet.ReadByteNext())        //Read in the requested player count
	if lobby.GetPlayersTooMany(clientPlayerCount, false) { //Check to see if there's enough open spots in the lobby
		return fmt.Errorf("unable to add %d players to lobby with %d/%d players", clientPlayerCount)
	}

	protocolVersion := int(packet.ReadByteNext()) //Read in the client's protocol version
	if protocolVersion != 25 {                    //We currently only support Stick Fight v25
		return fmt.Errorf("protocol version %d is unsupported", protocolVersion)
	}

	newClient := NewClient(lobby, packet.Src, steamID, clientPlayerCount) //Create a new client to host the new players
	lobby.ClientAdd(newClient)                                            //Add the new client to the lobby's client list

	//Initialize the client
	packetClientInit := NewPacket(packetTypeClientInit, 0, 0) //Create a clientInit packet to initialize this client
	packetClientInit.Grow(8 + int64(lobby.CurrentLevel.Size()))
	packetClientInit.WriteByteNext(0x1)                                 //Set to 1 to accept the connection, anything else to refuse the connection
	packetClientInit.WriteByteNext(byte(newClient.Players[0].Index))    //Set to the first new playerIndex that will be used by this client
	packetClientInit.WriteByteNext(byte(lobby.MaxPlayers))              //Set to the maximum amount of players for this lobby
	packetClientInit.WriteByteNext(lobby.CurrentLevel.Type())           //Set the map type of the current level
	packetClientInit.WriteI32LENext([]int32{lobby.CurrentLevel.Size()}) //Set the map size of the current level
	packetClientInit.WriteBytesNext(lobby.CurrentLevel.Data())          //Set the map data of the current level

	lobbyPlayers := lobby.GetPlayers()
	for i := 0; i < len(lobbyPlayers); i++ {
		packetClientInit.Grow(8)
		if lobbyPlayers[i] != nil {
			packetClientInit.WriteU64LENext([]uint64{lobbyPlayers[i].Client.SteamID.ID})
			if lobbyPlayers[i].Client.SteamID.ID != 0 && lobbyPlayers[i].Client.Addr.String() != packet.Src.String() {
				packetClientInit.Grow(52)
				pStats := lobbyPlayers[i].Stats
				packetClientInit.WriteI32LENext([]int32{
					pStats.Wins, pStats.Kills, pStats.Deaths, pStats.Suicides, pStats.Falls,
					pStats.CrownSteals,
					pStats.BulletsHit, pStats.BulletsMissed, pStats.BulletsShot,
					pStats.Blocks, pStats.PunchesLanded,
					pStats.WeaponsPickedUp, pStats.WeaponsThrown,
				})
			}
		} else {
			packetClientInit.WriteU64LENext([]uint64{0})
		}
	}

	//TODO: Weapons
	packetClientInit.Grow(2)
	packetClientInit.WriteU16LENext([]uint16{0}) //How many weapons to spawn

	//Lobby settings
	packetClientInit.Grow(4)
	packetClientInit.WriteBytesNext([]byte{
		0, //Still not entirely sure, gets assigned to OptionsHolder.maps on the client and no issues when set to 0
		lobby.Health,
		lobby.Regen,
		2, //Set weapon spawn rate to 2 so clients don't request to spawn weapons
	})

	//Send the clientInit packet!
	lobby.Server.SendPacket(packetClientInit, packet.Src)
	log.Info("Initialized client ", packet.Src, " for ", clientPlayerCount, " players")

	//Send the workshop map cycle to the client
	lobby.WorkshopMapsLoaded(packet.Src)

	return nil
}

//ClientAdd adds the specified client to the lobby
func (lobby *Lobby) ClientAdd(client *Client) {
	if lobby.GetPlayersTooMany(client.GetPlayerCount(), false) {
		return
	}

	if len(lobby.Clients) == 0 {
		lobby.LobbyOwner = client.SteamID
		lobby.Server.SendPacket(NewPacket(packetTypeRequestingOptions, 0, 0), client.Addr)
	}

	//Add the client to the list of available clients
	lobby.Clients = append(lobby.Clients, client)

	//Initialize each of the players in the client
	for clientPlayer := 0; clientPlayer < client.GetPlayerCount(); clientPlayer++ {
		playerIndex := lobby.GetNextPlayerIndex()
		client.Players[clientPlayer].Index = playerIndex             //Set the next player index for this player
		lobby.ClientJoined(client.Addr, playerIndex, client.SteamID) //Tell the lobby that this client has joined
	}
}

//ClientRemoveByClientIndex removes the specified client from the lobby
func (lobby *Lobby) ClientRemoveByClientIndex(clientIndex int) {
	if !lobby.IsRunning() {
		return
	}

	//Make sure this client actually exists
	if clientIndex < 0 || clientIndex >= len(lobby.Clients) {
		return
	}

	//Get the SteamID of the client
	steamID := lobby.Clients[clientIndex].SteamID

	//Close the client
	lobby.Clients[clientIndex].Close()

	//Remove the client from the lobby
	lobby.Clients[clientIndex] = nil                                 //Nullify the client
	copy(lobby.Clients[clientIndex:], lobby.Clients[clientIndex+1:]) //Shift every client after this client left by one
	lobby.Clients = lobby.Clients[:len(lobby.Clients)-1]             //Remove the last element

	if len(lobby.Clients) > 0 {
		lobby.ClientLeft(steamID) //Tell the other players that this client left
	} else {
		lobby.Close() //Close the lobby, since there's no more players
	}
}

//ClientJoined broadcasts to the lobby that the specified player is now part of this lobby
func (lobby *Lobby) ClientJoined(addr *net.UDPAddr, playerIndex int, steamID CSteamID) {
	packetClientJoined := NewPacket(packetTypeClientJoined, 0, 0)
	packetClientJoined.Grow(9)
	packetClientJoined.WriteByteNext(byte(playerIndex))
	packetClientJoined.WriteU64LENext([]uint64{steamID.ID})
	lobby.BroadcastPacket(packetClientJoined, addr)
	log.Info("Client ", steamID, " joined the lobby!")
}

//ClientLeft broadcasts to the lobby that the specified SteamID is no longer part of this lobby
func (lobby *Lobby) ClientLeft(steamID CSteamID) {
	packetClientLeft := NewPacket(packetTypeClientLeft, 0, 0)
	packetClientLeft.SteamID = steamID
	lobby.BroadcastPacket(packetClientLeft, nil)
	log.Info("Client ", steamID, " left the lobby!")

	if lobby.LobbyOwner.CompareCSteamID(steamID) {
		lobbyPlayers := lobby.GetPlayers()
		if len(lobbyPlayers) > 0 {
			lobby.LobbyOwner = lobby.GetPlayers()[0].Client.SteamID
			log.Info("New lobby owner: ", lobby.LobbyOwner)
		} else {
			lobby.LobbyOwner = NewCSteamID(0)
		}
	}
}

//WorkshopMapsLoaded sends the workshop map cycle to the specified client, or broadcasts if nil
func (lobby *Lobby) WorkshopMapsLoaded(addr *net.UDPAddr) {
	workshopMaps := make([]uint64, 0)
	for i := 0; i < len(lobby.Levels); i++ {
		if lobby.Levels[i].Type() == 2 {
			workshopMaps = append(workshopMaps, lobby.Levels[i].steamWorkshopID)
		}
	}
	for i := 0; i < len(lobbyLevels); i++ {
		if lobbyLevels[i].Type() == 2 {
			workshopMaps = append(workshopMaps, lobbyLevels[i].steamWorkshopID)
		}
	}
	if len(workshopMaps) > 0 {
		packetWorkshopMapsLoaded := NewPacket(packetTypeWorkshopMapsLoaded, 1, 0)
		packetWorkshopMapsLoaded.Grow(2 + int64(len(workshopMaps)*8)) //Grow by 2 bytes for workshop map count, then 8 bytes per map
		packetWorkshopMapsLoaded.WriteU16LENext([]uint16{uint16(len(workshopMaps))})
		packetWorkshopMapsLoaded.WriteU64LENext(workshopMaps)

		if addr != nil {
			lobby.Server.SendPacket(packetWorkshopMapsLoaded, addr)
		} else {
			lobby.BroadcastPacket(packetWorkshopMapsLoaded, nil)
		}
	}
}

//SpawnPlayer spawns the specified player at the specified coordinates
func (lobby *Lobby) SpawnPlayer(index int, posX, posY, rotX, rotY float32) {
	if !lobby.IsRunning() {
		return
	}

	clientIndex, playerIndex := lobby.GetIndexesByPlayerIndex(index)
	if clientIndex < 0 || playerIndex < 0 {
		log.Error("Unknown player ", index)
		return
	}

	if lobby.Clients[clientIndex].Players[playerIndex].Spawned {
		log.Warn("Ignoring spawn request for already spawned player ", index)
		return
	}

	flag := 0 //0 (default) = revive player for new map, 1 = forced die for spawned player
	if !lobby.CurrentLevel.IsLobby() && lobby.GetPlayerCount(true) > 1 {
		flag = 1
	}

	packetClientSpawned := NewPacket(packetTypeClientSpawned, 0, 0)
	packetClientSpawned.Grow(26)
	packetClientSpawned.WriteByteNext(byte(index)) //Write the player index
	packetClientSpawned.WriteF32LENext([]float32{
		posX, posY, 0,
		rotX, rotY, 0,
	})
	packetClientSpawned.WriteByteNext(byte(flag))

	lobby.Clients[clientIndex].Players[playerIndex].Spawned = true

	lobby.BroadcastPacket(packetClientSpawned, nil)
	log.Info("Spawned player ", index, " at position {X:", posX, ",Y:", posY, "} with rotation {X:", posX, ",Y:", posY, "} using flag ", flag)
}

//ReadyUp marks a player as ready
func (lobby *Lobby) ReadyUp(packet *Packet) {
	if !lobby.IsRunning() {
		return
	}

	playerCount := int(packet.ReadByteNext())
	for i := 0; i < playerCount; i++ {
		playerIndex := int(packet.ReadByteNext())
		clientIndex, clientPlayerIndex := lobby.GetIndexesByPlayerIndex(playerIndex)
		if clientIndex <= -1 || clientPlayerIndex <= -1 {
			continue
		}
		if !lobby.Clients[clientIndex].Paused { //If the client is marked as paused, don't accept the automatic ready-up
			lobby.Clients[clientIndex].Players[clientPlayerIndex].Ready = true
		}
	}

	if lobby.MatchInProgress() {
		lobby.Server.SendPacket(NewPacket(packetTypeStartMatch, 0, 0), packet.Src)
	} else {
		go lobby.StartMatch()
	}
}

//StartMatch starts the match if all players are ready
func (lobby *Lobby) StartMatch() {
	if !lobby.IsRunning() {
		return
	}

	if lobby.CurrentLevel.IsLobby() {
		log.Warn("Can't start match on lobby map!")
		return
	}

	if lobby.MatchInProgress() {
		log.Warn("Can't start match when already in fight!")
		return
	}

	notReady := false
	players := lobby.GetPlayers()
	for _, player := range players {
		if player != nil && !player.Ready {
			lobby.PlayerSaid(player.Index, "I'm not ready!")
			lobby.PlayerThought(player.Index, "If you can't ready up,\ntry typing /ready")
			notReady = true
			break
		}
	}

	if notReady {
		log.Warn("Can't start match until all players are ready!")
		return
	}

	time.Sleep(time.Second * 3)

	//TODO: Send list of pre-spawned weapons
	//TODO: Start goroutines for each object to track

	//Initialize the map
	lobby.InitMap()

	lobby.FightStartTime = time.Now()
	lobby.BroadcastPacket(NewPacket(packetTypeStartMatch, 0, 0), nil)
	log.Info("Started match!")

	lastWeaponSpawn := time.Now()
	weaponSpawnWait := randomizer.Intn(lobby.WeaponSpawnRateMax-lobby.WeaponSpawnRateMin) + lobby.WeaponSpawnRateMin
	if lobby.TourneyRules {
		weaponSpawnWait = randomizer.Intn(5-3) + 3 //3s min, 5s max
	}
	for lobby.MatchInProgress() {
		if !lobby.MatchInProgress() {
			break
		}

		if int(time.Now().Sub(lastWeaponSpawn)/time.Second) >= weaponSpawnWait {
			lobby.SpawnWeaponRandom()

			weaponSpawnWait = randomizer.Intn(lobby.WeaponSpawnRateMax-lobby.WeaponSpawnRateMin) + lobby.WeaponSpawnRateMin
			if lobby.TourneyRules {
				weaponSpawnWait = randomizer.Intn(5-3) + 3
			}
			lastWeaponSpawn = time.Now()
		}
	}
}

//MatchInProgress returns true if the match is in progress
func (lobby *Lobby) MatchInProgress() bool {
	lobby.Lock()         //Try to lock this check, so someone else can change it in sequence instead of it be ignored
	defer lobby.Unlock() //Unlock after we return the value
	return !lobby.FightStartTime.IsZero()
}

//UnReadyAllPlayers unreadies every player
func (lobby *Lobby) UnReadyAllPlayers() {
	if !lobby.IsRunning() {
		return
	}

	for i := 0; i < len(lobby.Clients); i++ {
		if lobby.Clients[i].GetPlayerCount() > 0 {
			for j := 0; j < len(lobby.Clients[i].Players); j++ {
				lobby.Clients[i].Players[j].Ready = false
				lobby.Clients[i].Players[j].Health = lobby.GetMaxHealth()
			}
		}
	}
}

//CheckWinner checks to see if the specified playerIndex is the winner, and starts a new match if they are
func (lobby *Lobby) CheckWinner() {
	if !lobby.IsRunning() {
		return
	}

	if !lobby.CurrentLevel.IsLobby() {
		if !lobby.MatchInProgress() {
			return
		}
	}

	if lobby.CheckingWinner {
		return
	}
	lobby.CheckingWinner = true

	survivors := make([]*Player, 0)

	for _, pl := range lobby.GetPlayers() {
		if pl != nil {
			if pl.Health > 0 {
				survivors = append(survivors, pl)
			}
		}
	}

	log.Trace("-----\n\nPlayers: ", lobby.GetPlayers(), "\n\nSurvivors: ", survivors, "\n\n")

	if len(survivors) == 1 {
		log.Info("Player ", survivors[0].Index, " is the winner!")
		lobby.ChangeMap(-1, survivors[0].Index)
	}

	if len(survivors) == 0 {
		log.Info("No one survived!")
		lobby.ChangeMap(-1, 255)
	}

	lobby.CheckingWinner = false
}

//ChangeMap changes the map and declares the winner
func (lobby *Lobby) ChangeMap(mapIndex, winnerIndex int) {
	if !lobby.IsRunning() {
		return
	}

	/*
		If a real winnerIndex is specified, and the match hasn't even started yet (which is prior to every client sending clientReadyUp),
		that means that a glitch occurred somewhere with the map change and caused it to be initiated twice, which can cause lag due to
		double map loads when the server broadcasts both, because a real map change should only occur once while the match is still in
		progress (as this function will end the match), and also because if you use the /map command to change the map before the last
		match could begin, the winnerIndex will be set to 255, which means no one won!

		The solution: If the match was ended already, and the winnerIndex is a real player who won somehow before the next match started,
		don't allow the map change.
	*/
	if !lobby.CurrentLevel.IsLobby() {
		if !lobby.MatchInProgress() && winnerIndex != 255 {
			return
		}
	}

	lobby.FightStartTime = time.Time{}
	lobby.UnReadyAllPlayers()

	lobby.CompletedLevelsSinceLastStats++

	if mapIndex < 0 || mapIndex >= len(lobby.Levels) {
		if !lobby.TourneyRules && lobby.CompletedLevelsSinceLastStats >= 30 {
			lobby.CompletedLevelsSinceLastStats = 0
			lobby.CurrentLevel = newLevelLandfall(102)
		} else {
			lobby.CurrentLevel = lobby.Levels[randomizer.Intn(len(lobby.Levels)-1)]
		}
	} else {
		lobby.CurrentLevel = lobby.Levels[mapIndex]
	}

	packetMapChange := NewPacket(packetTypeMapChange, 0, 0)
	packetMapChange.Grow(2)
	packetMapChange.WriteByteNext(byte(winnerIndex))
	packetMapChange.WriteByteNext(lobby.CurrentLevel.Type())
	packetMapChange.Grow(int64(lobby.CurrentLevel.Size()))
	packetMapChange.WriteBytesNext(lobby.CurrentLevel.Data())

	lobby.BroadcastPacket(packetMapChange, nil)
	log.Info("Changed map: ", lobby.CurrentLevel)
}

//TempMap assigns a temporary Landfall map to the fight
func (lobby *Lobby) TempMap(sceneIndex int32, winnerIndex int) {
	if !lobby.IsRunning() {
		return
	}

	lobby.FightStartTime = time.Time{}
	lobby.UnReadyAllPlayers()

	lobby.CurrentLevel = newLevelLandfall(sceneIndex)

	packetMapChange := NewPacket(packetTypeMapChange, 0, 0)
	packetMapChange.Grow(2)
	packetMapChange.WriteByteNext(byte(winnerIndex))
	packetMapChange.WriteByteNext(lobby.CurrentLevel.Type())
	packetMapChange.Grow(int64(lobby.CurrentLevel.Size()))
	packetMapChange.WriteBytesNext(lobby.CurrentLevel.Data())

	lobby.BroadcastPacket(packetMapChange, nil)
	log.Info("Changed map temporarily: ", lobby.CurrentLevel)
}

//InitMap initializes the map before it can begin
func (lobby *Lobby) InitMap() {
	if !lobby.IsRunning() {
		return
	}

	//Change the map scaling
	if lobby.CurrentLevel.MapSize > 0 {
		lobby.ChangeMapSize(lobby.CurrentLevel.MapSize)
	} else {
		lobby.ChangeMapSize(1)
	}

	//Initialize the ground weapons
	lobby.GroundWeaponsInit()

	//Initialize the map objects
	//lobby.MapObjectsInit()
}

//ChangeMapSize is called when the map size changes, so that anything which needs to scale can be scaled
func (lobby *Lobby) ChangeMapSize(newSize float32) {
	if !lobby.IsRunning() {
		return
	}

	lobby.LastAppliedScale = newSize / 10.0
}

//GroundWeaponsInit reads the current level's list of placed weapons and tells the connected clients that they're pre-spawned
func (lobby *Lobby) GroundWeaponsInit() {
	if !lobby.IsRunning() {
		return
	}

	placedWeapons := lobby.CurrentLevel.PlacedWeapons
	if len(placedWeapons) > 0 {
		packetGroundWeaponsInit := NewPacket(packetTypeGroundWeaponsInit, 0, 0)
		packetGroundWeaponsInit.Grow(2 + int64(len(placedWeapons)*12)) //Grow by 2 bytes for count, then 12 bytes per weapon
		packetGroundWeaponsInit.WriteU16LENext([]uint16{uint16(len(placedWeapons))})
		for i := 0; i < len(placedWeapons); i++ {
			weapon := placedWeapons[i]
			packetGroundWeaponsInit.WriteF32LENext([]float32{weapon.PositionX, weapon.PositionY})
			packetGroundWeaponsInit.WriteU16LENext([]uint16{lobby.GetNextWeaponSpawnID(false), lobby.GetNextObjectSpawnID(true)})
		}

		lobby.BroadcastPacket(packetGroundWeaponsInit, nil)
		log.Debug("Initialized ground weapons: ", placedWeapons)
	}
}

//PlayerUpdate syncs a player's network position and weapon
func (lobby *Lobby) PlayerUpdate(packet *Packet) { //420 IQ level strats here, buckle up
	if !lobby.IsRunning() {
		return
	}

	if !lobby.CurrentLevel.IsLobby() {
		if !lobby.MatchInProgress() {
			return
		}
	}

	clientIndex, client := lobby.GetClientByAddr(packet.Src) //Get the client index that this playerUpdate packet is from
	if client == nil {
		return
	}

	//The update channel is calculated as (playerIndex * 2) + 2, so reverse it to get the playerIndex
	playerIndex := (packet.Channel - 2) / 2
	if playerIndex <= -1 || playerIndex >= lobby.MaxPlayers { //Return if it's not a valid playerIndex
		return
	}

	//Get the client's player index by finding the client that holds a player with the matching playerIndex
	_, clientPlayerIndex := lobby.GetIndexesByPlayerIndex(playerIndex)

	//Make sure we aren't a damn fool
	if clientIndex <= -1 || clientPlayerIndex <= -1 {
		return
	}

	//Send the playerUpdate packet out before processing it, rip if it's invalid
	lobby.BroadcastPacket(packet, packet.Src)

	netPosition := NetworkPosition{
		Position:     Vector2{float32(packet.ReadI16LENext(1)[0]), float32(packet.ReadI16LENext(1)[0])}, //Read in the position of the player
		Rotation:     Vector2{float32(packet.ReadByteNext()), float32(packet.ReadByteNext())},           //Read in the rotation axis of the player
		YValue:       int(packet.ReadByteNext()),                                                        //Read in the player's YValue (known to be 100 for holding the up key, 156 for holding the down key, unknown for controllers)
		MovementType: MovementType(packet.ReadByteNext()),                                               //Read in the movement type of the player
	}

	netWeapon := NetworkWeapon{
		FightState: FightState(packet.ReadByteNext()), //Read in the fight state of the player
	}

	projectileCount := packet.ReadU16LENext(1)[0]           //Read in the amount of projectiles to read
	projectiles := make([]Projectile, int(projectileCount)) //We max out at 256 projectiles, for now...
	if len(projectiles) > 0 {                               //If we have projectiles available to read
		for i := 0; i < len(projectiles); i++ { //Loop over the projectiles that we can store
			//Read in the data about the projectile
			projectiles[i].ShootPosition = Vector2{float32(packet.ReadI16LENext(1)[0]), float32(packet.ReadI16LENext(1)[0])}
			projectiles[i].Shoot = Vector2{float32(packet.ReadByteNext()), float32(packet.ReadByteNext())}
			projectiles[i].SyncIndex = packet.ReadU16LENext(1)[0]
		}
	}
	netWeapon.Projectiles = projectiles
	if projectileCount > 256 { //If we maxed out at 256 projectiles and have more to be read
		packet.SeekByte((int64(projectileCount)*8)-int64(256), true) //Seek ahead so that the weapon type can be correctly read
		//TODO: Store pages of projectiles or find an alternative indexing that supports uint16 or int16 indexes
	}
	netWeapon.WeaponType = WeaponType(packet.ReadByteNext()) //Read in the player's current weapon type

	//Here's the strat
	lobby.Clients[clientIndex].Players[clientPlayerIndex].Position = netPosition
	lobby.Clients[clientIndex].Players[clientPlayerIndex].Weapon = netWeapon

	if logPlayerUpdate { //It's really spammy, trust me
		log.Debug(
			"Player ", playerIndex, ": ",
			"Position(", netPosition.Position, ") Rotation(", netPosition.Rotation, ") YValue:", netPosition.YValue, " Movement: ", netPosition.MovementType,
			" Fight:", netWeapon.FightState, " WeaponType:", netWeapon.WeaponType,
			" Projectiles:", projectiles,
		)
	}
}

//PlayerTookDamage syncs a player willingly admitting that they took damage
func (lobby *Lobby) PlayerTookDamage(packet *Packet) {
	if !lobby.IsRunning() {
		return
	}

	if !lobby.CurrentLevel.IsLobby() {
		if !lobby.MatchInProgress() {
			return
		}
	}

	_, client := lobby.GetClientByAddr(packet.Src) //Get the client index that this playerUpdate packet is from
	if client == nil {
		return
	}

	//The update channel is calculated as (playerIndex * 2) + 2, so reverse it to get the playerIndex
	playerIndex := (packet.Channel - 2) / 2
	if playerIndex <= -1 || playerIndex >= lobby.MaxPlayers { //Return if it's not a valid playerIndex
		return
	}

	//Get the client's player index by finding the client that holds a player with the matching playerIndex
	clientIndex, clientPlayerIndex := lobby.GetIndexesByPlayerIndex(playerIndex)

	//Make sure we aren't a damn fool
	if clientIndex <= -1 || clientPlayerIndex <= -1 {
		return
	}

	attackerIndex := int(packet.ReadByteNext())
	attackerClientIndex, attackerClientPlayerIndex := lobby.GetIndexesByPlayerIndex(attackerIndex)
	if attackerClientIndex <= -1 || attackerClientPlayerIndex <= -1 {
		return
	}

	if lobby.Clients[attackerClientIndex].Players[attackerClientPlayerIndex].Health <= 0 {
		return
	}

	damage := packet.ReadF32LENext(1)[0]
	particleDirection := Vector2{}
	if playParticles := packet.ReadByteNext(); playParticles == 1 {
		particleDirection.X = packet.ReadF32LENext(1)[0]
		particleDirection.Y = packet.ReadF32LENext(1)[0]
	}
	damageType := damageTypeOther
	if packet.ByteOffset() < packet.ByteCapacity() {
		damageType = DamageType(packet.ReadByteNext())
	}

	//Make sure this player isn't already dead
	if lobby.Clients[clientIndex].Players[clientPlayerIndex].Health <= 0 {
		log.Warn("Player ", playerIndex, " took damage despite being dead!")
		//return
	}

	//Make sure the player is ready if this isn't the lobby map
	if !lobby.Clients[clientIndex].Players[clientPlayerIndex].Ready && !lobby.CurrentLevel.IsLobby() {
		log.Warn("Player ", playerIndex, " took damage despite not being ready!")
		return
	}

	if damage == 666.666 {
		log.Info("Player ", playerIndex, " took a killing blow from player ", attackerIndex, " of type ", damageType)

		//Kill the targeted player
		lobby.Clients[clientIndex].Players[clientPlayerIndex].Health = 0
		lobby.Clients[clientIndex].Players[clientPlayerIndex].Stats.Deaths++

		//Give the attacker a kill
		if attackerIndex != playerIndex {
			lobby.Clients[attackerClientIndex].Players[attackerClientPlayerIndex].Stats.Kills++
		}

		//Broadcast the damage
		lobby.BroadcastPacket(packet, packet.Src)

		//Check for the winner
		lobby.CheckWinner()
		return
	}

	log.Info("Player ", playerIndex, " took ", damage, " damage from player ", attackerIndex, " of type ", damageType)

	//Remove the specified health from the player
	lobby.Clients[clientIndex].Players[clientPlayerIndex].Health -= damage

	//Broadcast the damage
	lobby.BroadcastPacket(packet, packet.Src)
	//lobby.BroadcastPacket(packet, nil)

	if lobby.Clients[clientIndex].Players[clientPlayerIndex].Health <= 0 {
		lobby.CheckWinner()
	}
}

//PlayerFallOut syncs a player falling out of bounds
func (lobby *Lobby) PlayerFallOut(packet *Packet) {
	if !lobby.IsRunning() {
		return
	}

	if !lobby.CurrentLevel.IsLobby() {
		if !lobby.MatchInProgress() {
			return
		}
	}

	//The update channel is calculated as (playerIndex * 2) + 2, so reverse it to get the playerIndex
	playerIndex := (packet.Channel - 2) / 2
	if playerIndex <= -1 || playerIndex >= lobby.MaxPlayers { //Return if it's not a valid playerIndex
		return
	}

	//Get the client's player index by finding the client that holds a player with the matching playerIndex
	clientIndex, clientPlayerIndex := lobby.GetIndexesByPlayerIndex(playerIndex)

	//Make sure we aren't a damn fool
	if clientIndex <= -1 || clientPlayerIndex <= -1 {
		return
	}

	if len(lobby.GetPlayers()) == 1 {
		lobby.ChangeMap(-1, 255)
		return
	}

	lobby.Clients[clientIndex].Players[clientPlayerIndex].Health = 0
	lobby.Clients[clientIndex].Players[clientPlayerIndex].Stats.Deaths++

	//Broadcast the fallout
	lobby.BroadcastPacket(packet, packet.Src)
	//lobby.BroadcastPacket(packet, nil)

	lobby.CheckWinner()
}

//PlayerTalked syncs a player's chat message and processes chat commands
func (lobby *Lobby) PlayerTalked(packet *Packet) {
	if !lobby.IsRunning() {
		return
	}

	//The event channel is calculated as (playerIndex * 2) + 2 + 1, so reverse it to get the playerIndex
	playerIndex := (packet.Channel - 1 - 2) / 2
	if playerIndex <= -1 || playerIndex >= lobby.MaxPlayers { //Return if it's not a valid playerIndex
		return
	}

	//Get the client index and the client's player index by finding the client that holds a player with the matching playerIndex
	clientIndex, clientPlayerIndex := lobby.GetIndexesByPlayerIndex(playerIndex)

	//Make sure we aren't a damn fool
	if clientIndex <= -1 || clientPlayerIndex <= -1 {
		return
	}

	//Read in the message
	msg := string(packet.Bytes())
	if lobby.Server.HasSwear(msg) {
		lobby.PlayerThought(playerIndex, "No swearing!")
		return
	}

	//Broadcast the message
	lobby.BroadcastPacket(packet, packet.Src)

	//Log it
	log.Trace("[CHAT:", lobby.Clients[clientIndex].SteamID.ID, "] ", lobby.Clients[clientIndex].SteamID.GetUsername(), ": ", msg)

	if string(msg[0]) == "/" {
		cmd := strings.Split(string(msg[1:]), " ")
		switch cmd[0] {
		case "options":
			lobby.Server.SendPacket(NewPacket(packetTypeRequestingOptions, 0, 0), packet.Src)

		case "ping":
			delay := uint32(time.Now().Unix()) - packet.Timestamp
			lobby.PlayerSaid(playerIndex, "%d second(s)\n2+ is bad", int(delay))

		case "public":
			if lobby.IsOwner(lobby.Clients[clientIndex].SteamID) {
				lobby.Public = true
				lobby.PlayerSaid(playerIndex, "Set lobby to public!")
			} else {
				lobby.PlayerSaid(playerIndex, "No permissions!")
			}
		case "private":
			if lobby.IsOwner(lobby.Clients[clientIndex].SteamID) {
				lobby.Public = false
				lobby.PlayerSaid(playerIndex, "Set lobby to private!")
			} else {
				lobby.PlayerSaid(playerIndex, "No permissions!")
			}

		case "pause", "unready", "afk", "brb":
			lobby.Clients[clientIndex].Paused = true
			lobby.PlayerSaid(playerIndex, "Paused for next match!")
		case "resume", "ready":
			lobby.Clients[clientIndex].Paused = false
			for i := 0; i < len(lobby.Clients[clientIndex].Players); i++ {
				lobby.Clients[clientIndex].Players[i].Ready = true
			}
			lobby.PlayerSaid(playerIndex, "Ready!")

			if !lobby.MatchInProgress() {
				lobby.StartMatch()
			}

		case "tourney", "tournament", "challenge", "hard", "hardmode":
			if lobby.IsOwner(lobby.Clients[clientIndex].SteamID) {
				lobby.TourneyRules = !lobby.TourneyRules
				if lobby.TourneyRules {
					lobby.PlayerSaid(playerIndex, "Enabled tournament rules!")
				} else {
					lobby.PlayerSaid(playerIndex, "Disabled tournament rules!")
				}
			} else {
				lobby.PlayerSaid(playerIndex, "No permissions!")
			}

		case "hp":
			if len(cmd) < 2 {
				lobby.PlayerSaid(playerIndex, "HP: %.2f", lobby.Clients[clientIndex].Players[clientPlayerIndex].Health)
				break
			}

			if !lobby.IsOwner(lobby.Clients[clientIndex].SteamID) {
				lobby.PlayerSaid(playerIndex, "No permissions!")
				break
			}

			healthBytes := []byte(cmd[1])
			if healthBytes[0] < 0 || healthBytes[0] > 6 {
				lobby.PlayerSaid(playerIndex, "Invalid HP setting!")
				break
			}

			lobby.Health = healthBytes[0]
			lobby.PlayerSaid(playerIndex, "Set max HP: %.2f", lobby.GetMaxHealth())

		case "maxplayers":
			if !lobby.IsOwner(lobby.Clients[clientIndex].SteamID) {
				lobby.PlayerSaid(playerIndex, "No permissions!")
				break
			}

			if len(cmd) < 2 {
				lobby.PlayerSaid(playerIndex, "/maxplayers playerCount")
				break
			}

			maxPlayers, err := strconv.Atoi(cmd[1])
			if err != nil {
				lobby.PlayerSaid(playerIndex, "Invalid playerCount!")
				break
			}

			if maxPlayers < lobby.MaxPlayers {
				lobby.PlayerSaid(playerIndex, "Cannot lower max players yet!")
				break
			}

			lobby.MaxPlayers = maxPlayers
			lobby.PlayerSaid(playerIndex, "Set max players to %d!", maxPlayers)

		case "travel":
			if len(cmd) < 3 {
				lobby.PlayerSaid(playerIndex, "/travel posX posY")
				break
			}

			posX, err := strconv.Atoi(cmd[1])
			if err != nil {
				lobby.PlayerSaid(playerIndex, "Invalid posX!")
				break
			}
			posY, err := strconv.Atoi(cmd[2])
			if err != nil {
				lobby.PlayerSaid(playerIndex, "Invalid posY!")
				break
			}

			packetPlayerUpdate := NewPacket(packetTypePlayerUpdate, lobby.Clients[clientIndex].Players[clientPlayerIndex].GetChannelUpdate(), lobby.Clients[clientIndex].SteamID.ID)
			packetPlayerUpdate.Grow(12)
			packetPlayerUpdate.WriteI16LENext([]int16{int16(posX), int16(posY)})
			packetPlayerUpdate.WriteBytesNext(make([]byte, 8))

			lobby.BroadcastPacket(packetPlayerUpdate, nil)
			lobby.PlayerSaid(playerIndex, "Traveled to X:%d Y:%d", posX, posY)

		case "map":
			if len(cmd) < 2 {
				lobby.PlayerSaid(playerIndex, "Current map: %s", lobby.CurrentLevel)
				break
			}

			if !lobby.IsOwner(lobby.Clients[clientIndex].SteamID) {
				lobby.PlayerSaid(playerIndex, "No permissions!")
				break
			}

			switch cmd[1] {
			case "add":
				if len(cmd) < 4 {
					lobby.PlayerSaid(playerIndex, "/map add {landfall/steam} mapID")
					break
				}
				switch cmd[2] {
				case "landfall", "Landfall", "lf", "LF":
					mapIndex, err := strconv.Atoi(cmd[3])
					if err != nil || mapIndex < 0 {
						lobby.PlayerSaid(playerIndex, "Invalid map index!")
						break
					}
					lfMap := newLevelLandfall(int32(mapIndex))
					lobby.Levels = append(lobby.Levels, lfMap)
					lobby.PlayerSaid(playerIndex, "Added map: %s", lfMap)
				case "steam", "Steam", "workshop", "Workshop", "sw", "SW":
					workshopID, err := strconv.ParseUint(cmd[3], 10, 64)
					if err != nil {
						lobby.PlayerSaid(playerIndex, "Invalid workshop ID!")
						break
					}
					steamMap := newLevelCustomOnline(workshopID)
					lobby.Levels = append(lobby.Levels, steamMap)
					lobby.PlayerSaid(playerIndex, "Added map: %s", steamMap)

					//Broadcast the workshop map cycle
					lobby.WorkshopMapsLoaded(nil)
				default:
					lobby.PlayerSaid(playerIndex, "Unknown map type: %s", cmd[2])
					break
				}
			case "scene":
				if len(cmd) < 3 {
					lobby.PlayerSaid(playerIndex, "Must specify sceneIndex!")
					break
				}
				sceneIndex, err := strconv.Atoi(cmd[2])
				if err != nil || sceneIndex < 0 {
					lobby.PlayerSaid(playerIndex, "Invalid scene index!")
					break
				}
				lobby.TempMap(int32(sceneIndex), 255)
				lobby.PlayerSaid(playerIndex, "New map: Landfall %d!", sceneIndex)
			default:
				mapIndex, err := strconv.Atoi(cmd[1])
				if err != nil || mapIndex >= len(lobby.Levels) || mapIndex < -1 {
					lobby.PlayerSaid(playerIndex, "Invalid map index!\n0 to %d\n-1 for random", len(lobby.Levels)-1)
					break
				}
				lobby.ChangeMap(mapIndex, 255)
				lobby.PlayerSaid(playerIndex, "New map: %s!", lobby.CurrentLevel)
			}

		default:
			lobby.PlayerSaid(playerIndex, "Unknown command!")
		}
	}
}

//PlayerSaid pretends a player said something out loud
func (lobby *Lobby) PlayerSaid(playerIndex int, msg string, data ...interface{}) {
	if !lobby.IsRunning() {
		return
	}

	clientIndex, clientPlayerIndex := lobby.GetIndexesByPlayerIndex(playerIndex)
	if clientIndex <= -1 || clientPlayerIndex <= -1 {
		return
	}

	resp := NewPacket(packetTypePlayerTalked, lobby.Clients[clientIndex].Players[clientPlayerIndex].GetChannelEvent(), lobby.Clients[clientIndex].SteamID.ID)
	respBytes := []byte(fmt.Sprintf(msg, data...))
	resp.Grow(int64(len(respBytes)))
	resp.WriteBytesNext(respBytes)
	lobby.BroadcastPacket(resp, nil)

	log.Trace("#[CHAT:", lobby.Clients[clientIndex].SteamID.ID, "] ", lobby.Clients[clientIndex].SteamID.GetUsername(), ": ", string(respBytes))
}

//PlayerThought pretends a player said something to themselves, where no one else can hear them
func (lobby *Lobby) PlayerThought(playerIndex int, msg string, data ...interface{}) {
	if !lobby.IsRunning() {
		return
	}

	clientIndex, clientPlayerIndex := lobby.GetIndexesByPlayerIndex(playerIndex)
	if clientIndex <= -1 || clientPlayerIndex <= -1 {
		return
	}

	resp := NewPacket(packetTypePlayerTalked, lobby.Clients[clientIndex].Players[clientPlayerIndex].GetChannelEvent(), lobby.Clients[clientIndex].SteamID.ID)
	respBytes := []byte(fmt.Sprintf(msg, data...))
	resp.Grow(int64(len(respBytes)))
	resp.WriteBytesNext(respBytes)
	lobby.Server.SendPacket(resp, lobby.Clients[clientIndex].Addr)

	log.Trace("#[CHAT:", lobby.Clients[clientIndex].SteamID.ID, "] ", lobby.Clients[clientIndex].SteamID.GetUsername(), ": ", string(respBytes))
}

//SpawnWeapon spawns the specified weapon on the map
func (lobby *Lobby) SpawnWeapon(weaponID int, weaponSpawnPos Vector3) {
	if !lobby.IsRunning() {
		return
	}

	if !lobby.MatchInProgress() {
		return
	}

	nextWeaponSpawnID := lobby.GetNextWeaponSpawnID(false)
	nextObjectSpawnID := lobby.GetNextObjectSpawnID(false)

	packetWeaponSpawned := NewPacket(packetTypeWeaponSpawned, 0, 0)
	packetWeaponSpawned.Grow(8)
	packetWeaponSpawned.WriteByteNext(byte(weaponID))
	packetWeaponSpawned.WriteBytesNext([]byte{byte(weaponSpawnPos.Y), byte(weaponSpawnPos.Z)})
	packetWeaponSpawned.WriteU16LENext([]uint16{nextWeaponSpawnID, nextObjectSpawnID})
	if lobby.CurrentLevel.IsLobby() && lobby.CurrentLevel.IsStats() && lobby.CurrentLevel.sceneIndex >= 104 && lobby.CurrentLevel.sceneIndex <= 124 {
		packetWeaponSpawned.WriteByteNext(1)
	}

	lobby.BroadcastPacket(packetWeaponSpawned, nil)
	log.Info("Spawned weapon ", weaponID, " at position ", weaponSpawnPos)
}

//SpawnWeapons spawns a list of weapons at the specified positions on the map
func (lobby *Lobby) SpawnWeapons(weaponIDs []int, weaponSpawnPositions []Vector3) {
	if !lobby.IsRunning() {
		return
	}

	if !lobby.MatchInProgress() {
		return
	}

	if len(weaponIDs) != len(weaponSpawnPositions) {
		return
	}

	for i := 0; i < len(weaponIDs); i++ {
		lobby.SpawnWeapon(weaponIDs[i], weaponSpawnPositions[i])
	}
}

//SpawnWeaponRandom spawns a random weapon on the map
func (lobby *Lobby) SpawnWeaponRandom() {
	if !lobby.IsRunning() {
		return
	}

	if !lobby.MatchInProgress() {
		return
	}

	weaponIDs := make([]int, randomizer.Intn(lobby.GetPlayerCount(false)+1))
	weaponSpawnPositions := make([]Vector3, len(weaponIDs))
	for i := 0; i < len(weaponIDs); i++ {
		weaponIDs[i] = validWeapons[randomizer.Intn(len(validWeapons)-1)]

		height := 11.0 * lobby.LastAppliedScale
		x := float32(randomizer.Intn(8))
		if lobby.TourneyRules {
			x = float32(randomizer.Intn(2))
		}
		if lobby.LastSpawnedWeaponOnLeftSide {
			x *= -1.0
		}
		lobby.LastSpawnedWeaponOnLeftSide = !lobby.LastSpawnedWeaponOnLeftSide

		weaponSpawnPositions[i] = Vector3{0, 1 * height, 1 * x}
	}

	lobby.SpawnWeapons(weaponIDs, weaponSpawnPositions)
}
