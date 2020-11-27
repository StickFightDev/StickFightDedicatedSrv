package main

import (
	"encoding/binary"
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
	connQueue     map[string]bool //map[UDP address] ignore this queue entry
)

const (
	noSteamID = uint64(0)
)

func main() {
	randomizer = rand.New(rand.NewSource(time.Now().UnixNano()))
	connQueue = make(map[string]bool)

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
	addHandler(packetTypePlayerFallOut, onPlayerFallOut)
	addHandler(packetTypeStartMatch, onStartMatch)

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
			n, addr, err := srv.ReadFromUDP(buffer)
			if err != nil {
				log.Error(addr, ": ", err)
				continue
			}

			var connLobby *lobby
			for _, lobby := range lobbies {
				for _, lobbyPlayer := range lobby.Players {
					if lobbyPlayer.Addr.String() == addr.String() {
						connLobby = lobby
					}
				}
			}

			timestamp := binary.LittleEndian.Uint32(buffer[0:4]) //4 bytes at 0x0
			if timestamp < lastTimestamp {
				log.Warn("outdated packet, last timestamp was ", lastTimestamp, " and current timestamp is ", timestamp)
				//continue
			}
			lastTimestamp = timestamp

			pkTypeByte := buffer[4] //1 byte at 0x4
			pkType := packetType(pkTypeByte)
			switch pkType {
			case packetTypeClientRequestingAccepting:
				if ignore, ok := connQueue[addr.String()]; ok && !ignore {
					log.Warn("ignoring multiple client accepting requests from queued client ", addr.String())
					continue //We don't need to listen to multiple connection attempts from someone already queueing, it breaks things!
				}
				connQueue[addr.String()] = false
			}

			data := make([]byte, 0)
			dataLen := n - 5
			if dataLen > 0 {
				data = buffer[5 : dataLen+5] //X bytes at 0x5
			}

			pk := newPacket(pkType)
			if len(data) > 0 {
				pk.Grow(int64(len(data)))
				pk.WriteBytes(0, data)
			}
			pk.Src = addr
			log.Info("Handling packet: ", pk)
			go pk.Handle(connLobby)
		}
	}()

	log.Trace("Waiting for exit call")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT)
	<-sc

	log.Trace("SIGINT received!")
}
