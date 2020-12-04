package main

import (
	"encoding/binary"
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

			//Custom end of packet header, directly following the packet's data
			//Size so far: 9 byte
			//0x0 (8 bytes, uint64) - The Steam ID of the user who is the intended recipient of the packet
			//0x8 (1 byte,  int)    - The channel, which was originally handled by Steam's networking library, and is required by the game's packet handling logic

			//Read needed stuff from the custom end of packet header
			eophSize := 9
			targetSteamID := binary.LittleEndian.Uint64(buffer[len(buffer)-9 : len(buffer)-1])
			channel := int(buffer[len(buffer)-1])

			//Remove the end of packet header from the raw buffer, so that the data of the packet doesn't include it
			buffer = buffer[0 : len(buffer)-eophSize]

			//Get the timestamp of the packet's original manifestation
			timestamp := binary.LittleEndian.Uint32(buffer[0:4]) //4 bytes at 0x0
			if timestamp < lastTimestamp {                       //This packet is older than the most recent packet, so it is outdated and must be ignored
				log.Warn("outdated packet, last timestamp was ", lastTimestamp, " and current timestamp is ", timestamp)
				continue //Goodbye packet, maybe you'll be faster next time :c
			}
			//Update the last recorded packet timestamp
			lastTimestamp = timestamp

			//Get the type of the packet
			pkTypeByte := buffer[4]          //1 byte at 0x4
			pkType := packetType(pkTypeByte) //Translate from byte to packetType, which is really just a byte

			//Packet firewall
			/*switch pkType {
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
					log.Warn("packet type ", getPacketType(pkType), " not allowed, client ", addr.String(), " is still waiting to be queued")
					continue
				}
			}*/

			//Create a slice to store the data bytes
			data := make([]byte, 0)
			dataLen := n - 5 - eophSize //Calculate the length of the packet data
			if dataLen > 0 {            //If there's data
				data = buffer[5 : dataLen+5] //Read the data from 0x5
			}

			//Create the packet
			pk := newPacket(pkType, channel, targetSteamID) //Create a packet in memory to store our packet data in for handling
			if len(data) > 0 {                              //If there's data
				pk.Grow(int64(len(data))) //Grow the packet's internal buffer to be able to store the data
				pk.WriteBytes(0, data)    //Write the data into the packet's internal buffer starting at 0x0
			}
			pk.Timestamp = timestamp
			pk.Src = addr                    //Store the source address of the packet
			pk.TargetSteamID = targetSteamID //Store the target Steam ID of the packet

			log.Info("Handling packet: ", pk)
			go pk.Handle(connLobby) //Handle the packet in a goroutine
		}
	}()

	log.Trace("Waiting for exit call")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT)
	<-sc

	log.Trace("SIGINT received!")
}
