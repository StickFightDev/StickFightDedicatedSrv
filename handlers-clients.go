package main

func onPing(p *packet, l *lobby) {
	if p.ByteCapacity() != 0 {
		log.Debug("ping from ", p.Src, ", sending pong")
		l.SendTo(newPacket(packetTypePingResponse), p.Src)
	} else {
		log.Debug("ping from ", p.Src)
	}
}
func onPingResponse(p *packet, l *lobby) {
	log.Debug("pong from ", p.Src)
}

func onClientRequestingAccepting(p *packet, l *lobby) {
	log.Debug("trying to accept client ", p.Src)

	addedToLobby := false
	for _, l := range lobbies {
		playerIndex, err := l.AddPlayer(p.Src)
		if err == nil {
			addedToLobby = true
			log.Debug("added client ", p.Src, " to old lobby as player ", playerIndex)
			break
		}
	}
	if !addedToLobby {
		nl := newLobby()

		_, err := nl.AddPlayer(p.Src)
		if err != nil {
			log.Error("unable to add client ", p.Src, " to newly created lobby")
			return
		}

		lobbies = append(lobbies, nl)
		log.Debug("added client ", p.Src, " to newly created lobby as host")
	}
}

func onClientRequestingIndex(p *packet, l *lobby) {
	steamID := p.ReadU64LENext(1)[0]
	_ = l.KickPlayerSteamID(steamID) //Kick this player if they're still connected from a previous session

	playerIndex := l.GetPlayerIndex(p.Src)
	if playerIndex == -1 {
		packetClientInit := newPacket(packetTypeClientInit)
		packetClientInit.Grow(1)
		packetClientInit.WriteByteNext(0) //Set to != 1 to refuse connection
		l.SendTo(packetClientInit, p.Src)
		return
	}

	if steamID == 0 { //Safety net for running additional instances of the game, disable for production servers
		steamID = 1337 + uint64(playerIndex)
	}

	packetClientJoined := newPacket(packetTypeClientJoined)
	packetClientJoined.Grow(9)
	packetClientJoined.WriteByteNext(byte(playerIndex))
	packetClientJoined.WriteU64LENext([]uint64{steamID})

	l.Broadcast(packetClientJoined, p.Src)
	log.Debug("Told lobby that a new client is in town: ", packetClientJoined)

	l.SetPlayerSteamID(p.Src, steamID)

	packetClientInit := newPacket(packetTypeClientInit)

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
		log.Debug("Player: ", l.Players[i])

		if l.Players[i].SteamID != 0 && l.Players[i].Addr.String() != p.Src.String() {
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
	log.Info("Sent player index: ", packetClientInit)
}

func onClientRequestingToSpawn(p *packet, l *lobby) {
	playerIndex := int(p.ReadByteNext()) //Read the player index

	if realPlayerIndex := l.GetPlayerIndex(p.Src); realPlayerIndex != playerIndex {
		log.Warn("Player ", realPlayerIndex, " is requesting for player ", playerIndex, " to spawn")
	}

	if l.Players[playerIndex].Status.Spawned {
		log.Error("Player ", playerIndex, " has already spawned")
		return
	}

	position := vector3{
		X: p.ReadF32LENext(1)[0],
		Y: p.ReadF32LENext(1)[0],
		Z: p.ReadF32LENext(1)[0],
	}

	rotation := vector3{
		X: p.ReadF32LENext(1)[0],
		Y: p.ReadF32LENext(1)[0],
		Z: p.ReadF32LENext(1)[0],
	}

	l.SpawnPlayer(playerIndex, position, rotation)
}

func onClientReadyUp(p *packet, l *lobby) {
	playerIndex := p.ReadByte(0)

	for i := 0; i < int(playerIndex); i++ {
		playerIndex2 := int(p.ReadByte(int64(i + 1)))
		l.Players[playerIndex2].Status.Ready = true
	}

	if l.InFight {
		l.SendTo(newPacket(packetTypeStartMatch), l.Players[int(p.ReadByte(1))].Addr)
		return
	}

	l.TryStartMatch()
}
