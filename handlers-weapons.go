package main

func onClientRequestingWeaponDrop(p *packet, l *lobby) {
	nextWeaponSpawnID := l.GetNextWeaponSpawnID(false)
	nextSyncableObjectSpawnID := l.GetNextSyncableObjectSpawnID(false)

	p.Type = packetTypeWeaponDropped
	p.Grow(4)
	p.WriteU16LENext([]uint16{nextWeaponSpawnID, nextSyncableObjectSpawnID})
	l.Broadcast(p, nil)

	log.Info("Player ", l.GetPlayerIndex(p.Src), " dropped weapon ", nextWeaponSpawnID, "!")
}

func onClientRequestingWeaponPickUp(p *packet, l *lobby) {
	playerIndex := int(p.ReadByteNext())
	weaponSpawnID := p.ReadU16LENext(1)[0]

	if weapon, ok := l.SpawnedWeapons[weaponSpawnID]; ok && weapon != nil {
		p.Type = packetTypeWeaponWasPickedUp
		l.Broadcast(p, nil)
		log.Info("Player ", playerIndex, " picked up weapon ", weaponSpawnID, "!")
	} else {
		log.Warn("Player ", playerIndex, " tried to pick up invalid weapon ", weaponSpawnID, "!")
	}
}

func onClientRequestingWeaponThrow(p *packet, l *lobby) {
	nextWeaponSpawnID := l.GetNextWeaponSpawnID(false)
	nextSyncableObjectSpawnID := l.GetNextSyncableObjectSpawnID(false)

	p.Type = packetTypeWeaponThrown
	p.Grow(4)
	p.WriteU16LE(p.ByteCapacity()-4, []uint16{nextWeaponSpawnID, nextSyncableObjectSpawnID})
	//p.WriteU16LENext([]uint16{nextWeaponSpawnID, nextSyncableObjectSpawnID})
	l.Broadcast(p, nil)

	log.Info("Player ", l.GetPlayerIndex(p.Src), " threw weapon ", nextWeaponSpawnID, "!")
}
