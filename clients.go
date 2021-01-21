package main

import (
	"net"
)

//Client holds a session with a lobby
type Client struct {
	Lobby *Lobby //The lobby that's hosting this client

	//The actual client details
	Addr *net.UDPAddr
	//LastTick time.Time
	PingInMs float64
	Closed   bool

	//The players on this client
	SteamID CSteamID
	Players []*Player

	//Client session tracking
	Paused bool //If the player is marked as paused, will make the lobby ignore the player's automatic ready-up
}

//NewClient returns a new client
func NewClient(lobby *Lobby, addr *net.UDPAddr, steamID uint64, playerCount int) *Client {
	newClient := &Client{
		Lobby: lobby,
		Addr:  addr,
		//LastTick: time.Now(),
		SteamID: NewCSteamID(steamID),
		Players: make([]*Player, playerCount),
	}

	for i := 0; i < playerCount; i++ {
		newClient.Players[i] = &Player{
			Client: newClient,
			Index:  -1,
			Health: lobby.GetMaxHealth(),
		}
	}

	return newClient
}

//Close closes a client
func (client *Client) Close() {
	if client.Closed {
		return
	}
	//TODO: Send a kick packet to tell the player they were kicked
	client.Addr = nil
	client.PingInMs = 0
	client.SteamID = NewCSteamID(0)
	client.Players = nil
	client.Closed = true
}

//IsClosed returns if the client is closed
func (client *Client) IsClosed() bool {
	return client.Closed
}

//GetPlayerCount returns how many players are playing from this client
func (client *Client) GetPlayerCount() int {
	return len(client.Players)
}
