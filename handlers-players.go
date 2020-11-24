package main

func onPlayerTookDamage(p *packet, l *lobby) {
	playerIndex := l.GetPlayerIndex(p.Src)
	if l.Players[playerIndex].Status.Dead {
		return
	}

	attackerIndex := int(p.ReadByteNext())
	damage := p.ReadF32LENext(1)[0]
	killingBlow := false
	if damage == 666.666 {
		killingBlow = true
	}

	typeDamage := byte(damageTypeOther)

	playParticles := p.ReadByteNext()
	particleDirection := vector3{}
	if playParticles == 1 {
		particleDirection.X = p.ReadF32LENext(1)[0]
		particleDirection.Y = p.ReadF32LENext(1)[0]

		if p.ByteCapacity() > 14 {
			typeDamage = p.ReadByteNext()
		}
	} else if p.ByteCapacity() > 6 {
		typeDamage = p.ReadByteNext()
	}

	if damageType(typeDamage) == damageTypePunch && playerIndex != attackerIndex {
		l.Players[attackerIndex].Stats.PunchesLanded++
	}

	if killingBlow {
		log.Info("Player ", playerIndex, " took a killing blow from player ", attackerIndex, " of type ", typeDamage)
		l.Players[playerIndex].Status.Health = 0
		l.Players[playerIndex].Status.Dead = true
		l.Players[playerIndex].Stats.Deaths++

		l.Players[attackerIndex].Stats.Kills++
		l.CheckWinner(attackerIndex)
	} else {
		log.Info("Player ", playerIndex, " took ", damage, " damage from player ", attackerIndex, " of type ", typeDamage)
		l.Players[playerIndex].Status.Health -= damage
	}

	l.Broadcast(p, p.Src)
}
