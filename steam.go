package main

import "github.com/Philipp15b/go-steamapi"

const (
	steamKeyAPI = "8FAF40F156C0D7DAF869385A3FF4EE1C"
)

var (
	steamUsernames = make(map[uint64]string)
)

func steamUsername(steamID uint64) string {
	if username, ok := steamUsernames[steamID]; ok {
		return username
	}

	summaries, err := steamapi.GetPlayerSummaries([]uint64{steamID}, steamKeyAPI)
	if err != nil {
		return string(steamID)
	}

	if len(summaries) == 0 {
		return string(steamID)
	}

	steamUsernames[steamID] = summaries[0].PersonaName
	return summaries[0].PersonaName
}
