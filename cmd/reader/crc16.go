package main

var crcTable = makeTable()

func makeTable() []uint16 {
	var array [256]uint16

	for i := 0; i < 256; i++ {
		mask := uint16(i)
		for j := 0; j < 8; j++ {
			if mask&0x1 == 1 {
				mask = (mask >> 1) ^ 0xA001
			} else {
				mask >>= 1
			}
		}
		array[i] = mask
	}

	return array[:]
}

// crc16 calculates the crc16 of the input bytes and updates the given crc16 with this value.
func crc16(crc uint16, buf []byte) uint16 {
	for _, v := range buf {
		crc = crcTable[byte(crc)^v] ^ (crc >> 8)
	}
	return crc
}
