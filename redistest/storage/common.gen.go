package storage

import (
	"strings"
)

func CompoundIndex(indexes ...string) string {
	return strings.Join(indexes, "-")
}
