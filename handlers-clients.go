package main

func onPing(p *packet, l *lobby) {
	if p.ByteCapacity() != 0 {
		l.SendTo(newPacket(packetTypePingResponse, 0, p.SteamID), p.Src)
	}
}
func onPingResponse(p *packet, l *lobby) {
}

func onClientRequestingAccepting(p *packet, l *lobby) {
	packetClientAccepted := newPacket(packetTypeClientAccepted, 1, 0)
	l.SendTo(packetClientAccepted, p.Src)
}

func onClientRequestingIndex(p *packet, l *lobby) {
	playerIndex := 0
	steamID := p.ReadU64LENext(1)[0]

	if l == nil {
		for _, testLobby := range lobbies {
			//Kick this player if they're still connected from a previous session
			_ = l.KickPlayerSteamID(steamID)

			newPlayerIndex, err := testLobby.AddPlayer(p.Src)
			if err == nil {
				log.Debug("added client ", p.Src, " to old lobby as player ", newPlayerIndex)
				playerIndex = newPlayerIndex
				l = testLobby
				break
			}
		}

		if l == nil {
			l = newLobby()
			_, err := l.AddPlayer(p.Src)
			if err != nil {
				log.Error("unable to add client ", p.Src, " to newly created lobby: ", err)
				packetClientInit := newPacket(packetTypeClientInit, 0, 0)
				packetClientInit.Grow(1)
				l.SendTo(packetClientInit, p.Src)
				return
			}
			lobbies = append(lobbies, l)
			log.Debug("added client ", p.Src, " to newly created lobby as host")
		}
	} else {
		playerIndex = l.GetPlayerIndex(p.Src)
	}

	localPlayerCount := int(p.ReadByteNext())
	if localPlayerCount > 1 { //We don't support multiple local players yet!
		log.Error("unable to add client ", p.Src, " to lobby: only 1 local player per client")
		packetClientInit := newPacket(packetTypeClientInit, 0, 0)
		packetClientInit.Grow(1)
		l.SendTo(packetClientInit, p.Src)
		return
	}

	protocolVersion := int(p.ReadByteNext())
	if protocolVersion != 25 { //We don't support anything other than Stick Fight v25 right now!
		log.Error("unable to add client ", p.Src, " to lobby: version ", protocolVersion, " is not supported")
		packetClientInit := newPacket(packetTypeClientInit, 0, 0)
		packetClientInit.Grow(1)
		l.SendTo(packetClientInit, p.Src)
		return
	}

	packetClientJoined := newPacket(packetTypeClientJoined, 0, 0)
	packetClientJoined.Grow(9)
	packetClientJoined.WriteByteNext(byte(playerIndex))
	packetClientJoined.WriteU64LENext([]uint64{steamID})
	l.Broadcast(packetClientJoined, p.Src)

	l.SetPlayerSteamID(p.Src, steamID)

	packetClientInit := newPacket(packetTypeClientInit, 0, 0)

	packetClientInit.Grow(2)
	packetClientInit.WriteByteNext(0x1)               //Set to 1 to accept connection, 0 with no other data to refuse connection
	packetClientInit.WriteByteNext(byte(playerIndex)) //Current player position in player list

	packetClientInit.Grow(5)
	packetClientInit.WriteByteNext(l.GetMap().Type())
	packetClientInit.WriteI32LENext([]int32{l.GetMap().Size()})
	packetClientInit.Grow(int64(l.GetMap().Size()))
	packetClientInit.WriteBytesNext(l.GetMap().Data())

	for i := 0; i < len(l.Players); i++ {
		packetClientInit.Grow(8)
		packetClientInit.WriteU64LENext([]uint64{l.Players[i].SteamID})

		if l.Players[i].SteamID != 0 && l.Players[i].Addr.String() != p.Src.String() {
			log.Debug("Player: ", l.Players[i])
			packetClientInit.Grow(52)
			packetClientInit.WriteI32LENext([]int32{
				l.Players[i].Stats.Wins,
				l.Players[i].Stats.Kills,
				l.Players[i].Stats.Deaths,
				l.Players[i].Stats.Suicides,
				l.Players[i].Stats.Falls,
				l.Players[i].Stats.CrownSteals,
				l.Players[i].Stats.BulletsHit,
				l.Players[i].Stats.BulletsMissed,
				l.Players[i].Stats.BulletsShot,
				l.Players[i].Stats.Blocks,
				l.Players[i].Stats.PunchesLanded,
				l.Players[i].Stats.WeaponsPickedUp,
				l.Players[i].Stats.WeaponsThrown,
			})
		}
	}

	packetClientInit.Grow(6)
	packetClientInit.WriteU16LENext([]uint16{0}) //Weapons to spawn, none until weapon keys are understood
	packetClientInit.WriteBytesNext([]byte{
		0, //Map count
		0, //Health
		0, //Regen
		0, //Weapon spawn rate
	})

	l.SendTo(packetClientInit, p.Src)
}

func onClientRequestingToSpawn(p *packet, l *lobby) {
	playerIndex := int(p.ReadByteNext()) //Read the player index

	if realPlayerIndex := l.GetPlayerIndex(p.Src); realPlayerIndex != playerIndex {
		log.Warn("Player ", realPlayerIndex, " is requesting for player ", playerIndex, " to spawn")
		//return
	}

	position := vector2{
		X: p.ReadF32LENext(1)[0],
		Y: p.ReadF32LENext(1)[0],
	}
	_ = p.ReadF32LENext(1)

	rotation := vector2{
		X: p.ReadF32LENext(1)[0],
		Y: p.ReadF32LENext(1)[0],
	}
	_ = p.ReadF32LENext(1)

	l.SpawnPlayer(playerIndex, position, rotation)
}

func onClientReadyUp(p *packet, l *lobby) {
	/*checkCount := int(p.ReadByte(0))
	for i := 0; i < checkCount; i++ {
		playerIndex2 := int(p.ReadByte(int64(i + 1)))
		l.Players[playerIndex2].Status.Ready = true
	}

	if l.InFight {
		l.SendTo(newPacket(packetTypeStartMatch), l.Players[int(p.ReadByte(1))].Addr)
		return
	}*/

	l.Players[l.GetPlayerIndex(p.Src)].Status.Ready = true

	if l.InFight {
		l.SendTo(newPacket(packetTypeStartMatch, 0, 0), p.Src)
	} else {
		l.TryStartMatch()
	}
}

func onStartMatch(p *packet, l *lobby) {
	l.TryStartMatch()
}

func onKickPlayer(p *packet, l *lobby) {
	steamID := l.GetPlayer(p.Src).SteamID
	err := l.KickPlayerSteamID(steamID)
	if err != nil {
		log.Error("Error kicking player ", steamUsername(steamID))
	} else {
		log.Info("Kicked player ", steamUsername(steamID))
	}
}

func onClientRequestingWeaponDrop(p *packet, l *lobby) {
	//TODO: Generate values correctly
	nextWeaponSpawnID         := uint16(0) //l.GetNextWeaponSpawnID(beginFromEnd = false)
	nextSyncableObjectSpawnID := uint16(0) //l.GetNextSyncableObjectSpawnID(beginFromEnd = false)

	p.Type = packetTypeWeaponDropped
	p.WriteU16LENext([]uint16{nextWeaponSpawnID, nextSyncableObjectSpawnID})
	l.Broadcast(p, nil)
}
func onClientRequestingWeaponPickUp(p *packet, l *lobby) {
	p.Type = packetTypeWeaponWasPickedUp
	l.Broadcast(p, nil)
}
