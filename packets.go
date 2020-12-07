package main

import (
	"errors"
	"fmt"
	"net"
	"time"

	crunch "github.com/superwhiskers/crunch/v3"
)

var (
	packetHandlers map[packetType]func(*packet, *lobby) = make(map[packetType]func(*packet, *lobby))
)

const (
	sophSize = 5
	eophSize = 9
)

func addHandler(packetType packetType, handler func(*packet, *lobby)) {
	packetHandlers[packetType] = handler
}

type packet struct {
	*crunch.Buffer //Holds the data buffer, and provides additional methods to directly read and write on this buffer

	Timestamp uint32       //The timestamp of the packet
	Src       *net.UDPAddr //The source UDP socket where this packet came from, nil if sending a packet
	Channel   int          //The channel for this packet to travel through
	SteamID   uint64       //The Steam ID of the user who sent or is intended to receive this packet
	Type      packetType   //The type of the packet, to allow easy packet creation
}

func newPacket(pkType packetType, channel int, steamID uint64) *packet {
	pk := &packet{crunch.NewBuffer(make([]byte, 0)), uint32(time.Now().Unix()), nil, channel, steamID, pkType}
	return pk
}

func (p *packet) Handle(l *lobby) {
	//Tunnel the packet to a target client if specified
	if p.SteamID != 0 && l != nil {
		for _, pl := range l.Players {
			if pl.Addr != nil && pl.SteamID == p.SteamID {
				l.SendTo(p, pl.Addr)
				break
			}
		}
	}
	if handler, ok := packetHandlers[p.Type]; ok { //If p.Type (the packet type) is a key in the packetHandlers map
		p.SeekByte(0, false) //Make sure the handler starts at offset 0 regardless of pre-processing
		handler(p, l)        //Execute its value handler, which is a packet handler function that takes the source packet from the socket and its lobby as arguments
	} else {
		log.Error("packet type ", getPacketType(p.Type), " not implemented yet, data: ", p.AsBytes()) //We don't have a handler for this packet, so log it and its data as decimal bytes
	}

	//Mark this address as done processing the packet
	//addrQueue.done(p.Src)
}

func (p *packet) String() string {
	str := fmt.Sprintf("[%d %d]", p.Channel, int64(p.Timestamp))
	if p.SteamID != noSteamID {
		str += " " + steamUsername(p.SteamID)
	}
	str += fmt.Sprintf(": %s %v", getPacketType(p.Type), p.Bytes())

	return str
}

func (p *packet) AsBytes() []byte {
	buf := crunch.NewBuffer(make([]byte, sophSize+eophSize+int(p.ByteCapacity())))

	buf.WriteU32LENext([]uint32{p.Timestamp})
	buf.WriteByteNext(byte(p.Type))
	if p.ByteCapacity() > 0 {
		buf.WriteBytesNext(p.Bytes())
	}
	buf.WriteU64LENext([]uint64{p.SteamID})
	buf.WriteByteNext(byte(p.Channel))
	return buf.Bytes()
}

func newPacketFromBytes(data []byte) (*packet, error) {
	if len(data) < sophSize+eophSize {
		return nil, errors.New("packet size too small")
	}

	buf := crunch.NewBuffer(data)
	dataLen := int64(len(data) - sophSize - eophSize)

	//Official start of packet header
	//Size: 5 bytes + data
	//0x0  (4 bytes, uint32) - Packet timestamp
	//0x4  (1 byte,  int)    - Packet type
	//0x5… (x bytes)         - Packet data, to be interpreted by the packet type's handler
	p := newPacket(packetTypeNull, 0, 0)
	p.Timestamp = buf.ReadU32LENext(1)[0]
	p.Type = packetType(buf.ReadByteNext())
	if dataLen > 0 {
		p.Grow(dataLen)
		p.WriteBytesNext(buf.ReadBytesNext(dataLen))
	}

	//Custom end of packet header, directly following the packet's data
	//Size so far: 9 bytes
	//0x0 (8 bytes, uint64) - The Steam ID of the user who is the intended recipient of the packet
	//0x8 (1 byte,  int)    - The channel, which was originally handled by Steam's networking library, and is required by the game's packet handling logic
	p.SteamID = buf.ReadU64LENext(1)[0]
	p.Channel = int(buf.ReadByteNext())

	return p, nil
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
	packetTypeNull = 255
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
