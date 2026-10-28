package tools

import (
	"encoding/binary"
	"os"
)

func SaveToPcap(filePath string, packets []Packet) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	globalHeader := []uint32{0xa1b2c3d4, 0x00040002, 0x00000000, 0x00000000, 0x0000ffff, 0x00000001}
	for _, val := range globalHeader {
		if err := binary.Write(f, binary.LittleEndian, val); err != nil {
			return err
		}
	}

	for _, pkt := range packets {
		if len(pkt.Data) == 0 {
			continue
		}

		secs := uint32(pkt.Timestamp.Unix())
		usecs := uint32(pkt.Timestamp.Nanosecond() / 1000)
		length := uint32(len(pkt.Data))

		packetHeader := []uint32{secs, usecs, length, length}
		for _, val := range packetHeader {
			if err := binary.Write(f, binary.LittleEndian, val); err != nil {
				return err
			}
		}

		if _, err := f.Write(pkt.Data); err != nil {
			return err
		}
	}

	return nil
}