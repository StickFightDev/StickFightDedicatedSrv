package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	crunch "github.com/superwhiskers/crunch/v3"
)

var (
	packetHandlers map[packetType]func(*packet, *lobby) = make(map[packetType]func(*packet, *lobby))
)

func addHandler(packetType packetType, handler func(*packet, *lobby)) {
	packetHandlers[packetType] = handler
}

type packet struct {
	*crunch.Buffer

	Src  *net.UDPAddr
	Type packetType
}

func newPacket(pkType packetType) *packet {
	pk := &packet{crunch.NewBuffer(make([]byte, 0)), nil, pkType}
	return pk
}

func (p *packet) Handle(l *lobby) {
	if handler, ok := packetHandlers[p.Type]; ok {
		handler(p, l)
	} else {
		log.Error("packet type ", getPacketType(p.Type), " not implemented yet, data: ", p.AsBytes())
	}
}

func (p *packet) String() string {
	return fmt.Sprintf("[%s] %s: %v", p.Src, getPacketType(p.Type), p.Bytes())
}

func (p *packet) AsBytes() []byte {
	array := make([]byte, 0)
	timestamp := make([]byte, 4)
	binary.LittleEndian.PutUint32(timestamp, uint32(time.Now().Unix()))
	array = append(array, timestamp...)
	array = append(array, byte(p.Type))
	array = append(array, p.Bytes()...)
	return array
}

type packetType byte

const (
	packetTypePing packetType = iota
	packetTypePingResponse
	packetTypeClientJoined
	packetTypeClientRequestingAccepting
	packetTypeClientAccepted
	packetTypeClientInit
	packetTypeClientRequestingIndex
	packetTypeClientRequestingToSpawn
	packetTypeClientSpawned
	packetTypeClientReadyUp
	packetTypePlayerUpdate
	packetTypePlayerTookDamage
	packetTypePlayerTalked
	packetTypePlayerForceAdded
	packetTypePlayerForceAddedAndBlock
	packetTypePlayerLavaForceAdded
	packetTypePlayerFallOut
	packetTypePlayerWonWithRicochet
	packetTypeMapChange
	packetTypeWeaponSpawned
	packetTypeWeaponThrown
	packetTypeRequestingWeaponThrow
	packetTypeClientRequestWeaponDrop
	packetTypeWeaponDropped
	packetTypeWeaponWasPickedUp
	packetTypeClientRequestingWeaponPickUp
	packetTypeObjectUpdate
	packetTypeObjectSpawned
	packetTypeObjectSimpleDestruction
	packetTypeObjectInvokeDestructionEvent
	packetTypeObjectDestructionCollision
	packetTypeGroundWeaponsInit
	packetTypeMapInfo
	packetTypeMapInfoSync
	packetTypeWorkshopMapsLoaded
	packetTypeStartMatch
	packetTypeObjectHello
	packetTypeOptionsChanged
	packetTypeKickPlayer
)

func getPacketType(packetType packetType) string {
	switch packetType {
	case packetTypePing:
		return "ping"
	case packetTypePingResponse:
		return "pingResponse"
	case packetTypeClientJoined:
		return "clientJoined"
	case packetTypeClientRequestingAccepting:
		return "clientRequestingAccepting"
	case packetTypeClientAccepted:
		return "clientAccepted"
	case packetTypeClientInit:
		return "clientInit"
	case packetTypeClientRequestingIndex:
		return "clientRequestingIndex"
	case packetTypeClientRequestingToSpawn:
		return "clientRequestingToSpawn"
	case packetTypeClientSpawned:
		return "clientSpawned"
	case packetTypeClientReadyUp:
		return "clientReadyUp"
	case packetTypePlayerUpdate:
		return "playerUpdate"
	case packetTypePlayerTookDamage:
		return "playerTookDamage"
	case packetTypePlayerTalked:
		return "playerTalked"
	case packetTypePlayerForceAdded:
		return "playerForceAdded"
	case packetTypePlayerForceAddedAndBlock:
		return "playerForceAddedAndBlock"
	case packetTypePlayerLavaForceAdded:
		return "playerLavaForceAdded"
	case packetTypePlayerFallOut:
		return "playerFallOut"
	case packetTypePlayerWonWithRicochet:
		return "playerWonWithRicochet"
	case packetTypeMapChange:
		return "mapChange"
	case packetTypeWeaponSpawned:
		return "weaponSpawned"
	case packetTypeWeaponThrown:
		return "weaponThrown"
	case packetTypeRequestingWeaponThrow:
		return "requestingWeaponThrow"
	case packetTypeClientRequestWeaponDrop:
		return "clientRequestWeaponDrop"
	case packetTypeWeaponDropped:
		return "weaponDropped"
	case packetTypeWeaponWasPickedUp:
		return "weaponWasPickedUp"
	case packetTypeClientRequestingWeaponPickUp:
		return "clientRequestingWeaponPickUp"
	case packetTypeObjectUpdate:
		return "objectUpdate"
	case packetTypeObjectSpawned:
		return "objectSpawned"
	case packetTypeObjectSimpleDestruction:
		return "objectSimpleDestruction"
	case packetTypeObjectDestructionCollision:
		return "objectDestructionCollision"
	case packetTypeGroundWeaponsInit:
		return "groundWeaponsInit"
	case packetTypeMapInfo:
		return "mapInfo"
	case packetTypeMapInfoSync:
		return "mapInfoSync"
	case packetTypeWorkshopMapsLoaded:
		return "workshopMapsLoaded"
	case packetTypeStartMatch:
		return "startMatch"
	case packetTypeObjectHello:
		return "objectHello"
	case packetTypeOptionsChanged:
		return "optionsChanged"
	case packetTypeKickPlayer:
		return "kickPlayer"
	}

	return fmt.Sprintf("unknown(%d)", packetType)
}
