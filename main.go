package main

import (
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sync"
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
	connState     map[string]int //map[UDP address] state (waiting on accept, waiting on index, ready for all)
	addrQueue     addressQueue
)

const (
	noSteamID = uint64(0)
)

type addressQueue struct {
	sync.Mutex

	addresses map[string]bool //map[UDP address] packet is processing
}

func (aq *addressQueue) make() {
	if aq.addresses == nil {
		aq.addresses = make(map[string]bool)
	}
}
func (aq *addressQueue) queue(addr *net.UDPAddr) {
	aq.Lock()
	defer aq.Unlock()
	aq.make()
	aq.addresses[addr.String()] = true
}
func (aq *addressQueue) waiting(addr *net.UDPAddr) bool {
	aq.Lock()
	defer aq.Unlock()
	aq.make()
	return aq.addresses[addr.String()]
}
func (aq *addressQueue) done(addr *net.UDPAddr) {
	aq.Lock()
	defer aq.Unlock()
	aq.make()
	aq.addresses[addr.String()] = false
}

func main() {
	randomizer = rand.New(rand.NewSource(time.Now().UnixNano()))
	connState = make(map[string]int)

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

			/*//Wait until it's your turn again!
			if addrQueue.waiting(addr) {
				for {
					if !addrQueue.waiting(addr) {
						break
					}
				}
			}
			//Mark this address as processing a packet
			addrQueue.queue(addr)*/

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

			if pk.Timestamp < lastTimestamp { //This packet is older than the most recent packet, so it is outdated and must be ignored
				log.Warn("outdated packet from ", addr, ", last timestamp was ", lastTimestamp, " and packet timestamp is ", pk.Timestamp)
				continue //Goodbye packet, maybe you'll be faster next time :c
			}
			//Set the last timestamp to system time
			//lastTimestamp = uint32(time.Now().Unix())
			//Set the last timestamp
			lastTimestamp = pk.Timestamp

			//Packet firewall
			/*switch pk.Type {
			case packetTypePing, packetTypePingResponse:
				break
			case packetTypeClientRequestingAccepting:
				if state, ok := connState[addr.String()]; ok && state >= 1 {
					//log.Warn("ignoring multiple client requesting accepting packets from queued client ", addr.String())
					//continue
				}
				connState[addr.String()] = 1
			case packetTypeClientRequestingIndex:
				if state, ok := connState[addr.String()]; ok && state >= 2 {
					log.Warn("ignoring multiple client requesting index packets from queued client ", addr.String())
					continue
				}
				connState[addr.String()] = 2
			default:
				if connState[addr.String()] < 3 {
					log.Warn("packet type ", getPacketType(pk.Type), " not allowed, client ", addr.String(), " is still waiting to be queued")
					continue
				}
			}*/

			if pk.Type != packetTypePlayerUpdate {
				log.Debug("Handling packet from ", pk.Src, ": ", pk)
			}
			go pk.Handle(connLobby) //Handle the packet in a goroutine
		}
	}()

	log.Trace("Waiting for exit call")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT)
	<-sc

	log.Trace("SIGINT received!")
}
