package main

import (
	"math/rand"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/JoshuaDoes/logger"
)

var (
	log           = logger.NewLogger("sf:srv", 2)
	address       = "0.0.0.0:1337"
	maxBufferSize = 8192
	srv           *net.UDPConn
	randomizer    *rand.Rand
)

const (
	noSteamID = uint64(0)
)

func main() {
	randomizer = rand.New(rand.NewSource(time.Now().UnixNano()))

	s, err := net.ResolveUDPAddr("udp4", address)
	if err != nil {
		log.Fatal(err)
	}

	log.Debug("Registering packet handlers...")
	addHandler(packetTypePing, onPing)
	addHandler(packetTypePingResponse, onPingResponse)
	addHandler(packetTypeClientRequestingAccepting, onClientRequestingAccepting)
	addHandler(packetTypeClientRequestingIndex, onClientRequestingIndex)
	addHandler(packetTypeClientRequestingToSpawn, onClientRequestingToSpawn)
	addHandler(packetTypeClientReadyUp, onClientReadyUp)
	addHandler(packetTypePlayerUpdate, onPlayerUpdate)
	addHandler(packetTypePlayerTookDamage, onPlayerTookDamage)
	addHandler(packetTypePlayerTalked, onPlayerTalked)
	addHandler(packetTypePlayerForceAdded, onPlayerForceAdded)
	addHandler(packetTypePlayerForceAddedAndBlock, onPlayerForceAddedAndBlock)
	addHandler(packetTypePlayerLavaForceAdded, onPlayerLavaForceAdded)
	addHandler(packetTypePlayerFallOut, onPlayerFallOut)
	addHandler(packetTypeClientRequestingWeaponPickUp, onClientRequestingWeaponPickUp)
	addHandler(packetTypeClientRequestingWeaponDrop, onClientRequestingWeaponDrop)
	addHandler(packetTypeClientRequestingWeaponThrow, onClientRequestingWeaponThrow)
	addHandler(packetTypeStartMatch, onStartMatch)
	addHandler(packetTypeKickPlayer, onKickPlayer)

	log.Info("Listening on UDP socket ", s)
	srv, err = net.ListenUDP("udp4", s)
	if err != nil {
		log.Fatal(err)
	}
	defer srv.Close()

	log.Trace("Creating buffer with max size ", maxBufferSize)
	buffer := make([]byte, maxBufferSize)

	log.Debug("Spawning goroutine to read incoming packets")
	go func() {
		lastTimestamp := uint32(time.Now().Unix())

		for {
			//Reset the buffer
			buffer = make([]byte, maxBufferSize)

			//Read into the buffer
			n, addr, err := srv.ReadFromUDP(buffer)
			if err != nil {
				log.Error(addr, ": ", err)
				continue
			}

			//Determine if this UDP socket is part of a lobby already
			var connLobby *lobby
			for _, lobby := range lobbies {
				for _, lobbyPlayer := range lobby.Players {
					if lobbyPlayer.Addr.String() == addr.String() {
						connLobby = lobby
					}
				}
			}

			//Trim the buffer
			buffer = buffer[0:n]

			//Read the buffer into a packet
			pk, err := newPacketFromBytes(buffer)
			if err != nil {
				log.Error(err)
				continue //Goodbye false packet!
			}
			//Set the source address of the packet
			pk.Src = addr

			thread := true
			checkTime := true
			logHandle := true
			switch pk.Type {
			case packetTypePlayerUpdate:
				checkTime = false
				logHandle = false
			case packetTypePlayerTalked:
				checkTime = false
			case packetTypePlayerForceAdded:
				checkTime = false
			case packetTypePlayerForceAddedAndBlock:
				checkTime = false
			case packetTypePlayerLavaForceAdded:
				checkTime = false
			case packetTypePlayerFallOut:
				thread = false
				checkTime = false
			case packetTypePlayerWonWithRicochet:
				checkTime = false
			case packetTypePlayerTookDamage:
				thread = false
			}

			if checkTime {
				if pk.Timestamp < lastTimestamp { //This packet is older than the most recent packet, so it is outdated and must be ignored
					log.Warn("outdated packet from ", addr, ", last timestamp was ", lastTimestamp, " and packet timestamp is ", pk.Timestamp)
					continue //Goodbye packet, maybe you'll be faster next time :c
				}

				//Set the last timestamp
				lastTimestamp = pk.Timestamp
			}

			if logHandle {
				log.Debug("Handling packet from ", pk.Src, ": ", pk)
			}

			if thread {
				go pk.Handle(connLobby)
			} else {
				pk.Handle(connLobby)
			}
		}
	}()

	log.Trace("Waiting for exit call")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT)
	<-sc

	log.Trace("SIGINT received!")

	log.Info("Shutting down the server...")
	srv.Close()
}
