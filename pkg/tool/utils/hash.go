package utils

import (
	"fmt"
	"hash/crc32"
)

func CRC32Hash(input string) string {
	return fmt.Sprintf("%d", crc32.ChecksumIEEE([]byte(input)))
}
