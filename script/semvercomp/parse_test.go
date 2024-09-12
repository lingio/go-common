package main

import "testing"

func TestParseSomeVersions(t *testing.T) {
	cases := []struct {
		S string
		X semver
	}{
		{"v1.2.3", semver{1, 2, 3}},
		{"1.2.3-0.20240822132137-a8612a7f3238", semver{1, 2, 3}},
		{"1.30.2-0.20240822132137-a8612a7f3238", semver{1, 30, 2}},
	}
	for _, c := range cases {
		if x := parseSemver(c.S); x != c.X {
			t.Errorf("parse %s = %v != %v", c.S, c.X, x)
		}
	}
}

func TestCompareSomeVersions(t *testing.T) {
	cases := []struct {
		A semver
		B semver
		V int
	}{
		{semver{1, 2, 3}, semver{1, 2, 3}, 0},
		{semver{1, 2, 0}, semver{1, 2, 3}, -1},
		{semver{1, 30, 3}, semver{1, 30, 2}, 1},
	}
	for _, c := range cases {
		if v := c.A.Compare(c.B); v != c.V {
			t.Errorf("compare %v %v = %v != %v", c.A, c.B, v, c.V)
		}
	}

}
