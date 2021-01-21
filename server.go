package main

import (
	"fmt"
	"net"
	"runtime"
	"time"

	swearfilter "github.com/JoshuaDoes/gofuckyourself"
)

var (
	swears = []string{" "}
)

//Server holds a Stick Fight dedicated server
type Server struct {
	Addr string

	//Session
	Running bool
	Sock    *net.UDPConn
	Lobbies []*Lobby
	Filter  *swearfilter.SwearFilter
}

//NewServer returns a new server running on the specified UDP address
func NewServer(addr string) *Server {
	srv := &Server{
		Addr:    addr,
		Lobbies: make([]*Lobby, 0),
		Filter:  swearfilter.NewSwearFilter(true, swears...),
	}

	return srv
}

//IsRunning returns true if the server is currently running
func (srv *Server) IsRunning() bool {
	return srv.Running
}

//Close closes the server
func (srv *Server) Close() {
	if !srv.IsRunning() {
		return
	}

	log.Info("Closing server!")

	for _, lobby := range srv.Lobbies {
		lobby.Close()
	}

	srv.Sock.Close()
	srv.Running = false
}

//Run starts the server and ticks it until it's closed
func (srv *Server) Run() {
	if srv.Running {
		srv.Close()
	}

	udpAddr, err := net.ResolveUDPAddr("udp4", srv.Addr)
	if err != nil {
		log.Fatal("Unable to resolve UDP address for udp4 address ", srv.Addr)
	}
	log.Trace("Resolved UDP address for udp4 address ", srv.Addr)

	sock, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		log.Fatal("Unable to listen on UDP address ", udpAddr)
	}
	log.Trace("Listening on UDP address ", udpAddr)
	srv.Sock = sock

	srv.Running = true
	log.Info("Server is running!")

	for i := 0; i < runtime.NumCPU(); i++ {
		go srv.ReadPackets()
	}

	for srv.Running {
		if !srv.Running {
			break
		}

		time.Sleep(time.Millisecond * 10)
	}
}

//ReadPackets starts reading packets and handles them
func (srv *Server) ReadPackets() {
	buffer := make([]byte, maxBufferSize)

	for srv.Running {
		if !srv.Running {
			break
		}

		//Reset the buffer
		buffer = make([]byte, maxBufferSize)

		//Block until a packet is read into the buffer
		n, addr, err := srv.Sock.ReadFromUDP(buffer)
		if err != nil {
			log.Error(addr, ": ", err)
			continue
		}

		//Trim the buffer
		buffer = buffer[:n]

		//Handle the packet
		go srv.Handle(buffer, addr)
	}
}

//SendPacket sends a packet to a destination address
func (srv *Server) SendPacket(packet *Packet, addr *net.UDPAddr) {
	srv.Sock.WriteToUDP(packet.AsBytes(), addr)

	if packet.ShouldLog() {
		log.Trace("Sent to ", addr, ": ", packet)
	}
}

//Handle handles a packet for the server
func (srv *Server) Handle(buffer []byte, addr *net.UDPAddr) {
	//Read the buffer into a packet
	packet, err := NewPacketFromBytes(buffer)
	if err != nil {
		log.Error(err)
		return //Goodbye false packet!
	}

	//Set the source address of the packet
	packet.Src = addr

	//Log the packet
	if packet.ShouldLog() {
		log.Trace("Received from ", addr, ": ", packet)
	}

	if lobby := srv.GetLobbyByAddr(packet.Src); lobby != nil {
		lobby.Handle(packet)
		return
	}

	switch packet.Type {
	case packetTypePing:
		srv.ClientPong(packet.Src, packet.Bytes())

	case packetTypeClientRequestingAccepting:
		srv.ClientAccept(packet.Src)

	case packetTypeClientRequestingIndex:
		for _, lobby := range srv.Lobbies {
			//Try to initialize this client with the lobby
			err := lobby.ClientInit(packet)
			if err == nil {
				return
			}
		}

		lobby, err := NewLobby(srv)
		if err != nil {
			log.Error(err)
			srv.ClientReject(addr, err.Error())
			return
		}
		err = lobby.ClientInit(packet)
		if err != nil {
			log.Error(err)
			srv.ClientReject(addr, err.Error())
			return
		}
		srv.LobbyAdd(lobby)

	case packetTypeKickPlayer:
		//Just so we handle this if the client isn't in a lobby yet

	default:
		log.Error(fmt.Sprintf("Unhandled packet from %s: %s", packet.Src, packet))
	}
}

//ClientPong responds to a ping with a pong
func (srv *Server) ClientPong(addr *net.UDPAddr, data []byte) {
	packetPingResponse := NewPacket(packetTypePingResponse, 0, 0)
	if dataLen := int64(len(data)); dataLen > 0 {
		packetPingResponse.Grow(dataLen)
		packetPingResponse.WriteBytesNext(data)
	}
	srv.SendPacket(packetPingResponse, addr)
}

//ClientAccept accepts a client
func (srv *Server) ClientAccept(addr *net.UDPAddr) {
	packetClientAccepted := NewPacket(packetTypeClientAccepted, 1, 0)
	srv.SendPacket(packetClientAccepted, addr)
	log.Debug("Accepted client ", addr)
}

//ClientReject rejects a client
func (srv *Server) ClientReject(addr *net.UDPAddr, reason string) {
	packetClientInit := NewPacket(packetTypeClientInit, 0, 0)
	packetClientInit.Grow(1)
	if reason != "" {
		reasonBytes := []byte(reason)
		packetClientInit.Grow(int64(len(reasonBytes)))
		packetClientInit.WriteBytes(0x1, reasonBytes)
	}
	srv.SendPacket(packetClientInit, addr)
	if reason != "" {
		log.Debug("Rejected client ", addr, " with reason: ", reason)
	} else {
		log.Debug("Rejected client ", addr)
	}
}

//GetLobbyByAddr returns the lobby that the address is found in
func (srv *Server) GetLobbyByAddr(addr *net.UDPAddr) *Lobby {
	if len(srv.Lobbies) > 0 {
		for lobbyIndex := 0; lobbyIndex < len(srv.Lobbies); lobbyIndex++ {
			if srv.Lobbies[lobbyIndex].IsRunning() && len(srv.Lobbies[lobbyIndex].Clients) > 0 {
				for clientIndex := 0; clientIndex < len(srv.Lobbies[lobbyIndex].Clients); clientIndex++ {
					if srv.Lobbies[lobbyIndex].Clients[clientIndex].Addr.String() == addr.String() {
						//We found the lobby that this client is in!
						return srv.Lobbies[lobbyIndex]
					}
				}
			}
		}
	}

	//We didn't find the lobby, they must be new!
	return nil
}

//LobbyAdd adds the specified lobby to the server
func (srv *Server) LobbyAdd(lobby *Lobby) {
	srv.Lobbies = append(srv.Lobbies, lobby)
}

//HasSwear checks if the given message has a swear
func (srv *Server) HasSwear(message string) (tripped bool) {
	trippedWords, err := srv.Filter.Check(message)
	if err != nil {
		tripped = true
	}
	if len(trippedWords) > 0 {
		tripped = true
	}
	if tripped {
		log.Trace("[CHAT] Message has tripped words ", trippedWords, ": ", message)
	}
	return
}
