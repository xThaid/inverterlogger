package inverterlogger

import "github.com/howeyc/crc16"

func calcCheckSum8(bytes []byte) byte {
	sum := 0
	for _, b := range bytes {
		sum += int(b)
	}
	return byte(sum & 255)
}

func calcCRC16Modbus(bytes []byte) uint16 {
	return ^crc16.ChecksumIBM(bytes)
}
