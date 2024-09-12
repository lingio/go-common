package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func semver(v string) int {
	v = strings.TrimPrefix(v, "v") // v1.2.3 -> 1.2.3
	v, _, _ = strings.Cut(v, "-")  // 1.2.3-shasha -> 1.2.3

	parts := strings.Split(v, ".")
	var version int
	mult := 1
	for i := len(parts) - 1; i >= 0; i-- {
		val, err := strconv.Atoi(parts[i])
		if err != nil {
			fmt.Println(err, v)
			os.Exit(2)
		}
		version += val * mult
		mult *= 10
	}
	return version
}

func main() {
	if semver(os.Args[1]) >= semver(os.Args[2]) {
		os.Exit(0)
	}
	os.Exit(1)
}
