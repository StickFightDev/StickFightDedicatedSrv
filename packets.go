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
	*crunch.Buffer //Holds the data buffer, and provides additional methods to directly read and write on this buffer

	Timestamp     uint32       //The timestamp of the packet
	Src           *net.UDPAddr //The source UDP socket where this packet came from, nil if sending a packet
	Channel       int          //The channel for this packet to travel through
	SteamID       uint64       //The Steam ID of the user who sent this packet
	TargetSteamID uint64       //The Steam ID of the user intended to receive this packet
	Type          packetType   //The type of the packet, to allow easy packet creation
}

func newPacket(pkType packetType, channel int, targetSteamID uint64) *packet {
	pk := &packet{crunch.NewBuffer(make([]byte, 0)), uint32(time.Now().Unix()), nil, channel, 0, targetSteamID, pkType}
	return pk
}

func (p *packet) Handle(l *lobby) {
	if handler, ok := packetHandlers[p.Type]; ok { //If p.Type (the packet type) is a key in the packetHandlers map
		handler(p, l) //Execute its value handler, which is a packet handler function that takes the source packet from the socket and its lobby as arguments
	} else {
		log.Error("packet type ", getPacketType(p.Type), " not implemented yet, data: ", p.AsBytes()) //We don't have a handler for this packet, so log it and its data as decimal bytes
	}

	//Mark this address as done processing the packet
	//addrQueue.done(p.Src)
}

func (p *packet) String() string {
	return fmt.Sprintf("[@%s-%d %s] %s -> %s: %s %v", p.Src, p.Channel, time.Unix(int64(p.Timestamp), 0).String(), steamUsername(p.SteamID), steamUsername(p.TargetSteamID), getPacketType(p.Type), p.Bytes())
}

func (p *packet) AsBytes() []byte {
	//Holds the raw packet as bytes
	array := make([]byte, 0)

	timestamp := make([]byte, 4)
	binary.LittleEndian.PutUint32(timestamp, uint32(time.Now().Unix()))

	steamID := make([]byte, 8)
	binary.LittleEndian.PutUint64(steamID, p.SteamID)

	array = append(array, timestamp...)    //Timestamp of the packet
	array = append(array, byte(p.Type))    //Type of the packet
	array = append(array, p.Bytes()...)    //Packet data
	array = append(array, steamID...)      //Steam ID of the source user
	array = append(array, byte(p.Channel)) //Channel for packet to travel
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
