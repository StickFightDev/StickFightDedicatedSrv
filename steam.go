package main

import "github.com/Philipp15b/go-steamapi"

var (
	steamUsernames = make(map[uint64]string)
)

//CSteamID holds a Steam client ID and its username
type CSteamID struct {
	ID       uint64
	Username string
}

//NewCSteamID returns a new Steam client ID
func NewCSteamID(steamID uint64) CSteamID {
	clientID := CSteamID{
		ID: steamID,
	}

	return clientID
}

//GetUsername returns the username of the CSteamID
func (cSteamID CSteamID) GetUsername() string {
	if cSteamID.Username != "" {
		return cSteamID.Username
	}

	if steamUsername, ok := steamUsernames[cSteamID.ID]; ok {
		cSteamID.Username = steamUsername
		return steamUsername
	}

	summaries, err := steamapi.GetPlayerSummaries([]uint64{cSteamID.ID}, steamKey)
	if err != nil {
		return ""
	}

	if len(summaries) == 0 {
		return ""
	}

	steamUsernames[cSteamID.ID] = summaries[0].PersonaName
	cSteamID.Username = steamUsernames[cSteamID.ID]
	return cSteamID.Username
}

//CompareCSteamID evaluates if a CSteamID is the same as another
func (cSteamID CSteamID) CompareCSteamID(compareSteamID CSteamID) bool {
	return cSteamID.ID == compareSteamID.ID
}

//CompareSteamID evaluates if a CSteamID matches a SteamID
func (cSteamID CSteamID) CompareSteamID(steamID uint64) bool {
	return cSteamID.ID == steamID
}
