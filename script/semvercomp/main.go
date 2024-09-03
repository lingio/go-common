package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func semver(v string) int {
	if v[0] == 'v' {
		v = v[1:]
	}
	parts := strings.Split(v, ".")
	var version int
	mult := 1
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		val, err := strconv.Atoi(part)
		if err != nil {
			// do not compare things with strings in them, hashes etc.
			fmt.Println(fmt.Sprintf("Skipping non numeric part: %q in %q", part, v))
			continue
		} else {
			version += val * mult
			mult *= 10
		}
	}
	return version
}

func main() {
	if semver(os.Args[1]) >= semver(os.Args[2]) {
		os.Exit(0)
	}
	os.Exit(1)
}
