package utils

import (
	"fmt"
	"github.com/gosimple/slug"
	"hash/crc32"
)

func CRC32Hash(input string) string {
	return fmt.Sprintf("%d", crc32.ChecksumIEEE([]byte(input)))
}
func NameSlug(name string) string {
	return fmt.Sprintf(
		"%s-%s",
		CRC32Hash(name),
		slug.Make(name),
	)
}
