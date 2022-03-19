package main

import (
	"fmt"
	"os"
	"os/exec"
	"io/ioutil"

	"github.com/JoshuaDoes/json"
	"github.com/Philipp15b/go-steamapi"
)

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

//LoadWorkshopMaps updates and preloads a given list of Workshop maps in a batch command
func LoadWorkshopMaps(steamWorkshopIDs ...uint64) ([]*Level, error) {
	workshopItem := []string{"+workshop_download_item", "674940"}
	params := make([]string, 0)

	for i := 0; i < len(steamWorkshopIDs); i++ {
		id := fmt.Sprintf("%d", steamWorkshopIDs[i])
		log.Trace("Queueing workshop map ", id, " for integrity check")
		params = append(params, append(workshopItem, id)...)
	}

	log.Trace("Syncing workshop maps...")
	if err := scmd.Raw(params...); err != nil {
		return nil, err
	}

	for i := 0; i < len(steamWorkshopIDs); i++ {
		id := fmt.Sprintf("%d", steamWorkshopIDs[i])
		workshopMap := steamCmdDir + "/steamapps/workshop/content/674940/" + id + "/Level.bin"
		
		if _, err := os.Stat(workshopMap); os.IsNotExist(err) {
			return nil, err
		}
	}

	workshopMaps := make([]*Level, 0)
	for i := 0; i < len(steamWorkshopIDs); i++ {
		id := fmt.Sprintf("%d", steamWorkshopIDs[i])
		workshopMap := steamCmdDir + "/steamapps/workshop/content/674940/" + id + "/Level.bin"
		sfMap := "maps/" + id + ".json"

		if _, err := os.Stat(sfMap); os.IsNotExist(err) {
			log.Trace("Decoding workshop map ", id, "...")
			sfmu := exec.Command("SFMU", workshopMap, sfMap)
			if verbosityLevel == 2 {
				sfmu.Stdout = os.Stdout
				sfmu.Stderr = os.Stderr
			}
			if err := sfmu.Run(); err != nil {
				return nil, err
			}
			if _, err := os.Stat(sfMap); os.IsNotExist(err) {
				return nil, err
			}
		}

		log.Debug("Loading workshop map ", id, "...")
		mapJSON, err := ioutil.ReadFile(sfMap)
		if err != nil {
			return nil, err
		}

		m := &Level{}
		if err := json.Unmarshal(mapJSON, m); err != nil {
			return nil, err
		}
		workshopMaps = append(workshopMaps, m)
	}

	return workshopMaps, nil
}