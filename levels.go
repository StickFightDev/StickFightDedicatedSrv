package main

import (
	"encoding/binary"
	"fmt"

	crunch "github.com/superwhiskers/crunch/v3"
)

type level struct {
	mapType byte //0 = Landfall, 1 = Local, 2 = CustomOnline
	data    []byte

	sceneIndex      int32
	local           string
	steamWorkshopID uint64
}

func newMap(mapType byte, data []byte) *level {
	return &level{
		mapType: mapType,
		data:    data,
	}
}
func newMapLandfall(sceneIndex int32) *level {
	return &level{
		mapType:    0,
		sceneIndex: sceneIndex,
	}
}
func newMapLocal(path string) *level {
	return &level{
		mapType: 1,
		local:   path,
	}
}
func newMapCustomOnline(steamWorkshopID uint64) *level {
	return &level{
		mapType:         2,
		steamWorkshopID: steamWorkshopID,
	}
}

func (m *level) Type() byte {
	return m.mapType
}

func (m *level) Data() []byte {
	dataBuf := crunch.NewBuffer()

	switch m.mapType {
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

	return m.data //Return level data if unsupported handling
}

func (m *level) Size() int32 {
	switch m.mapType {
	case 0:
		return 4
	case 1:
		return int32(len(m.local))
	case 2:
		return 8
	}
	return int32(len(m.data))
}

func (m *level) String() string {
	switch m.mapType {
	case 0:
		return fmt.Sprintf("Landfall map: %d", int(binary.LittleEndian.Uint32(m.Data())))
	case 1:
		return string(m.Data()) + "/Level.bin"
	case 2:
		return fmt.Sprintf("Steam Workshop map: %v", binary.LittleEndian.Uint64(m.Data()))
	}
	return fmt.Sprintf("%d: %v", int(m.Type()), m.Data())
}
