package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type semver struct {
	major, minor, patch int
}

func (s semver) Compare(o semver) int {
	var (
		major = s.major - o.major
		minor = s.minor - o.minor
		patch = s.patch - o.patch
	)
	if major == 0 {
		if minor == 0 {
			return sign(patch)
		}
		return sign(minor)
	}
	return sign(major)
}

func sign(v int) int {
	if v > 0 {
		return 1
	} else if v < 0 {
		return -1
	} else {
		return 0
	}
}

func parseSemver(s string) semver {
	s = strings.TrimPrefix(s, "v") // v1.2.3 -> 1.2.3
	s, _, _ = strings.Cut(s, "-")  // 1.2.3-shasha -> 1.2.3

	parts := strings.Split(s, ".")
	var v [3]int
	for i := range v {
		val, err := strconv.Atoi(parts[i])
		if err != nil {
			fmt.Println(err, s)
			os.Exit(2)
		}
		v[i] = val
	}
	return semver{v[0], v[1], v[2]}
}

func main() {
	if parseSemver(os.Args[1]).Compare(parseSemver(os.Args[2])) >= 0 {
		os.Exit(0)
	}
	os.Exit(1)
}
