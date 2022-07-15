package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
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
	HTTP    *net.TCPListener
	Lobbies []*Lobby
	Filter  *swearfilter.SwearFilter
}

//Status holds server statistics
type Status struct {
	Address string `json:"address"`
	Online bool `json:"online"`
	Lobbies int `json:"lobbies"`
	MaxLobbies int `json:"maxLobbies"`
	Players int `json:"playersOnline"`
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

//Status returns the current server statistics
func (srv *Server) Status() *Status {
	players := 0
	for i := 0; i < len(srv.Lobbies); i++ {
		players += srv.Lobbies[i].GetPlayerCount(false)
	}

	return &Status{
		Address: srv.Addr,
		Online: srv.Running,
		Lobbies: len(srv.Lobbies),
		MaxLobbies: maxLobbies,
		Players: players,
	}
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
	srv.HTTP.Close()
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

	tcpAddr, err := net.ResolveTCPAddr("tcp", srv.Addr)
	if err != nil {
		log.Fatal("Unable to resolve TCP address for tcp address ", srv.Addr)
	}
	log.Trace("Resolved TCP address for tcp address ", srv.Addr)

	httpSock, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Fatal("Unable to listen on TCP address ", tcpAddr)
	}
	log.Trace("Listening on TCP address ", tcpAddr)
	srv.HTTP = httpSock

	srv.Running = true
	log.Info("Server is running!")

	for i := 0; i < runtime.NumCPU(); i++ {
		go srv.ReadPackets()
	}
	go srv.ReadHTTP()

	for srv.Running {
		if !srv.Running {
			break
		}

		time.Sleep(time.Millisecond * 1000)
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

//ReadHTTP starts reading HTTP packets and handles them
func (srv *Server) ReadHTTP() {
	buffer := make([]byte, maxBufferSize)

	for srv.Running {
		if !srv.Running {
			break
		}

		//Reset the buffer
		buffer = make([]byte, maxBufferSize)

		//Block until a client is encountered
		tcpConn, err := srv.HTTP.AcceptTCP()
		if err != nil {
			log.Error(srv.HTTP.Addr(), ": ", err)
			continue
		}
		log.Trace("Accepted TCP client: ", tcpConn.RemoteAddr())
		//defer tcpConn.Close()

		//Block until a packet is read into the buffer
		n, err := tcpConn.Read(buffer)
		if err != nil {
			log.Error(tcpConn.RemoteAddr(), ": ", err)
			continue
		}

		//Trim the buffer
		buffer = buffer[:n]

		packet, err := NewPacketFromBytes(buffer)
		if err != nil {
			log.Error("unable to create packet from bytes: ", err)
			continue
		}

		if packet.Type != packetTypeHTTP {
			log.Error("expected packet type HTTP over TCP")
			continue
		}

		/*resp, err := http.ReadResponse(bufio.NewReader(packet), nil)
		if err != nil {
			log.Error("unable to convert HTTP response: ", err)
			continue
		}*/
		resp := &http.Response{
			Status: "200 OK",
			StatusCode: 200,
			Proto: "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Body: io.NopCloser(bufio.NewReader(packet)),
			ContentLength: int64(len(packet.Bytes())),
		}

		/*n, err = tcpConn.Write(packet.Bytes())
		if err != nil {
			log.Error(tcpConn.RemoteAddr(), ": ", err)
			continue
		}*/

		err = resp.Write(tcpConn)
		if err != nil {
			log.Error(tcpConn.RemoteAddr(), ": ", err)
			continue
		}

		//tcpConn.CloseWrite()
		err = tcpConn.Close()
		log.Trace("Closed TCP client: ", err)
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
		log.Error("unable to create packet from bytes to handle: ", err)
		return //Goodbye false packet!
	}

	if packet.Type == packetTypeHTTP {
		return
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
		/* TODO: Move to srv.FindLobby(packet)
		for _, lobby := range srv.Lobbies {
			//Try to initialize this client with the lobby
			err := lobby.ClientInit(packet)
			if err == nil {
				return
			}
		}*/

		lobby, err := NewLobby(srv, "") //Create a new lobby with a random room code
		if err != nil {
			log.Error("unable to create new lobby: ", err)
			srv.ClientReject(addr, err.Error())
			return
		}
		err = lobby.ClientInit(packet)
		if err != nil {
			log.Error("unable to init client into new lobby: ", err)
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
		for i := 0; i < len(srv.Lobbies); i++ {
			if srv.Lobbies[i] != nil && srv.Lobbies[i].IsRunning() && len(srv.Lobbies[i].Clients) > 0 {
				for clientIndex := 0; clientIndex < len(srv.Lobbies[i].Clients); clientIndex++ {
					if srv.Lobbies[i].Clients[clientIndex].Addr.String() == addr.String() {
						//We found the lobby that this client is in!
						return srv.Lobbies[i]
					}
				}
			}
		}
	}

	//We didn't find the lobby, they must be new!
	return nil
}

//GetLobbyByCode returns the lobby matching the room code
func (srv *Server) GetLobbyByCode(code string) *Lobby {
	if len(srv.Lobbies) > 0 {
		for i := 0; i < len(srv.Lobbies); i++ {
			if srv.Lobbies[i] != nil && srv.Lobbies[i].IsRunning() && srv.Lobbies[i].LobbyRoomCode == code {
				return srv.Lobbies[i]
			}
		}
	}

	return nil
}

//LobbyAdd adds the specified lobby to the server
func (srv *Server) LobbyAdd(lobby *Lobby) {
	srv.Lobbies = append(srv.Lobbies, lobby)
}

//GetClientByAddr returns the client with a matching address
func (srv *Server) GetClientByAddr(addr *net.UDPAddr) (int, *Client) {
	for _, lobby := range srv.Lobbies {
		if lobby.Clients == nil || len(lobby.Clients) == 0 {
			continue
		}

		for clientIndex, client := range lobby.Clients {
			if client.Addr.String() == addr.String() {
				return clientIndex, client
			}
		}
	}
	return -1, nil
}

//GetClientBySteamID returns the client with a matching SteamID
func (srv *Server) GetClientBySteamID(steamID CSteamID) *Client {
	for _, lobby := range srv.Lobbies {
		if lobby.Clients == nil || len(lobby.Clients) == 0 {
			continue
		}

		for _, client := range lobby.Clients {
			if client.SteamID.CompareCSteamID(steamID) {
				return client
			}
		}
	}
	return nil
}

//GetClientBySteamUsername returns the client with a matching Steam username
func (srv *Server) GetClientBySteamUsername(steamUsername string) *Client {
	for _, lobby := range srv.Lobbies {
		if lobby.Clients == nil || len(lobby.Clients) == 0 {
			continue
		}

		for _, client := range lobby.Clients {
			if client.SteamID.GetUsername() == steamUsername {
				return client
			}

			if client.SteamID.GetNormalizedUsername() == steamUsername {
				return client
			}
		}
	}
	return nil
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
