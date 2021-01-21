package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"

	crunch "github.com/superwhiskers/crunch/v3"
)

var (
	defaultLevels = make([]*Level, 0)
	lobbyLevels   = make([]*Level, 0)
)

//Level holds a Stick Fight level
type Level struct {
	//Map identification
	levelType       byte   //0 = Landfall, 1 = Local, 2 = Workshop (CustomOnline), 3 = Streamed (CustomStream)
	data            []byte //The actual map ID or map data to be sent as a map identifier
	sceneIndex      int32  //If levelType == 0, the Unity scene index of this map
	local           string //If levelType == 1 || levelType == 3, the path to the map file
	steamWorkshopID uint64 //If levelType == 2, the Steam workshop ID of this map

	//The actual decoded map contents
	SpawnPoints   []Vector2         `json:"SpawnPoints"`   //The positions where each player can spawn, minimum 4
	PlacedObjects []*SyncableObject `json:"PlacedObjects"` //The pre-placed objects to sync
	PlacedWeapons []*SyncableWeapon `json:"PlacedWeapons"` //The pre-placed weapons to sync
	MapSize       float32           `json:"MapSize"`       //The map scaling to apply to some coordinates
	Theme         int               `json:"Theme"`         //The theme of the map
	Version       string            `json:"Version"`       //The version of Stick Fight used to create the map

	//Level session tracking
	SpawnedObjects map[uint16]*SyncableObject //A list of every spawned object to sync
	SpawnedWeapons map[uint16]*SyncableWeapon //A list of every spawned weapon to sync
}

func newLevel(levelType byte, data []byte) *Level {
	level := &Level{
		levelType: levelType,
		data:      data,
	}

	if err := level.Load(); err != nil {
		log.Error(err)
		return nil
	}

	return level
}
func newLevelLandfall(sceneIndex int32) *Level {
	level := &Level{
		levelType:  0,
		sceneIndex: sceneIndex,
	}

	return level
}
func newLevelLocal(path string) *Level {
	level := &Level{
		levelType: 1,
		local:     path,
	}

	if err := level.Load(); err != nil {
		log.Error(err)
		return nil
	}

	return level
}
func newLevelCustomOnline(steamWorkshopID uint64) *Level {
	level := &Level{
		levelType:       2,
		steamWorkshopID: steamWorkshopID,
	}

	if err := level.Load(); err != nil {
		log.Error(err)
		return nil
	}

	return level
}
func newLevelCustomStream(path string, data []byte) *Level {
	level := &Level{
		levelType: 3,
		local:     path,
		data:      data,
	}

	if err := level.Load(); err != nil {
		log.Error(err)
		return nil
	}

	return level
}

//Load loads the Stick Fight map into memory
func (m *Level) Load() error {
	//Initialize some fields
	m.SpawnedObjects = make(map[uint16]*SyncableObject)
	m.SpawnedWeapons = make(map[uint16]*SyncableWeapon)

	switch m.levelType {
	case 0:
		return errors.New("unable to load Landfall map")
	case 1:
		return errors.New("unable to load local map")
	case 2:
		workshopMap := steamCmdDir + "/steamapps/workshop/content/674940/" + strconv.Itoa(int(m.steamWorkshopID)) + "/Level.bin"
		if _, err := os.Stat(workshopMap); os.IsNotExist(err) {
			log.Trace("Downloading workshop item ", m.steamWorkshopID, "...")
			if err := scmd.DownloadWorkshopMod(674940, int(m.steamWorkshopID)); err != nil {
				return err
			}
		} else {
			log.Trace("Using pre-cached download for workshop item ", m.steamWorkshopID)
		}

		sfMap := "maps/" + strconv.Itoa(int(m.steamWorkshopID)) + ".json"
		if _, err := os.Stat(sfMap); os.IsNotExist(err) {
			log.Trace("Decoding workshop map ", m.steamWorkshopID, "...")
			sfmu := exec.Command("SFMU", workshopMap, sfMap)
			if verbosityLevel == 2 {
				sfmu.Stdout = os.Stdout
				sfmu.Stderr = os.Stderr
			}
			if err := sfmu.Run(); err != nil {
				return err
			}
			if _, err := os.Stat(sfMap); os.IsNotExist(err) {
				return err
			}
		} else {
			log.Trace("Using pre-decoded map for workshop item ", m.steamWorkshopID)
		}

		log.Trace("Loading workshop map ", m.steamWorkshopID, "...")
		mapJSON, err := ioutil.ReadFile(sfMap)
		if err != nil {
			return err
		}

		if err := json.Unmarshal(mapJSON, m); err != nil {
			return err
		}
	case 3:
		return errors.New("unable to load streamed map")
	default:
		return errors.New("unable to load unsupported map type")
	}

	for i := 0; i < len(m.PlacedObjects); i++ {
		m.SpawnedObjects[uint16(i)] = m.PlacedObjects[i]
	}
	for i := 0; i < len(m.PlacedWeapons); i++ {
		m.SpawnedWeapons[uint16(i)] = m.PlacedWeapons[i]
	}

	return nil
}

//Type returns the Stick Fight map type
func (m *Level) Type() byte {
	return m.levelType
}

//Data returns the Stick Fight map as bytes
func (m *Level) Data() []byte {
	dataBuf := crunch.NewBuffer()

	switch m.levelType {
	case 0:
		dataBuf.Grow(4)
		dataBuf.WriteI32LENext([]int32{m.sceneIndex})
		return dataBuf.Bytes()
	case 1:
		return []byte(m.local)
	case 2:
		dataBuf.Grow(8)
		dataBuf.WriteU64LENext([]uint64{m.steamWorkshopID})
		return dataBuf.Bytes()
	}

	return m.data //Return Level data if unsupported handling
}

//Size returns the size of the map data
func (m *Level) Size() int32 {
	switch m.levelType {
	case 0:
		return 4
	case 1:
		return int32(len(m.local))
	case 2:
		return 8
	}
	return int32(len(m.data))
}

func (m *Level) String() string {
	switch m.levelType {
	case 0:
		return fmt.Sprintf("Landfall map: %d", int(binary.LittleEndian.Uint32(m.Data())))
	case 1:
		return string(m.Data()) + "/Level.bin"
	case 2:
		return fmt.Sprintf("Steam Workshop map: %v", binary.LittleEndian.Uint64(m.Data()))
	case 3:
		return "Streamed map: " + m.local
	}
	return fmt.Sprintf("%d: %v", int(m.Type()), m.Data())
}

//IsLobby returns true if the map is a Landfall map with sceneIndex 0
func (m *Level) IsLobby() bool {
	switch m.levelType {
	case 0:
		if m.sceneIndex == 0 {
			return true
		}
	case 2:
		for i := 0; i < len(lobbyLevels); i++ {
			if m.steamWorkshopID == lobbyLevels[i].steamWorkshopID {
				return true
			}
		}
	}
	return false
}

//IsStats returns true if the map is a Landfall map with sceneIndex 102
func (m *Level) IsStats() bool {
	return m.levelType == 0 && m.sceneIndex == 102
}
