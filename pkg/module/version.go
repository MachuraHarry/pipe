package module

import (
	"strconv"
	"strings"
)

// semver holds the numeric components of a version string.
type semver struct {
	major, minor, patch int
	pre                 string
}

// parseSemver parses "major.minor.patch[-pre][+build]". A missing minor or
// patch defaults to 0. Invalid components make the version sort lower than
// any valid version.
func parseSemver(v string) semver {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexByte(v, '+'); i >= 0 {
		v = v[:i]
	}
	pre := ""
	if i := strings.IndexByte(v, '-'); i >= 0 {
		pre = v[i+1:]
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	nums := make([]int, 0, 3)
	for _, p := range parts {
		if p == "" {
			nums = append(nums, 0)
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			nums = append(nums, -1)
			continue
		}
		nums = append(nums, n)
	}
	for len(nums) < 3 {
		nums = append(nums, 0)
	}
	return semver{major: nums[0], minor: nums[1], patch: nums[2], pre: pre}
}

func compareSemver(a, b semver) int {
	if a.major != b.major {
		return cmpInt(a.major, b.major)
	}
	if a.minor != b.minor {
		return cmpInt(a.minor, b.minor)
	}
	if a.patch != b.patch {
		return cmpInt(a.patch, b.patch)
	}
	return cmpPre(a.pre, b.pre)
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func cmpPre(a, b string) int {
	if a == "" && b == "" {
		return 0
	}
	// No pre-release is greater than a pre-release.
	if a == "" {
		return 1
	}
	if b == "" {
		return -1
	}
	// Numeric identifiers compare numerically, alphanumeric lexically.
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		an, aErr := strconv.Atoi(aParts[i])
		bn, bErr := strconv.Atoi(bParts[i])
		aNumeric := aErr == nil
		bNumeric := bErr == nil
		switch {
		case aNumeric && bNumeric && an != bn:
			return cmpInt(an, bn)
		case aNumeric != bNumeric:
			// Numeric identifiers have lower precedence.
			if aNumeric {
				return -1
			}
			return 1
		case aParts[i] != bParts[i]:
			if aParts[i] < bParts[i] {
				return -1
			}
			return 1
		}
	}
	return cmpInt(len(aParts), len(bParts))
}

// versionLTE reports whether a <= b under SemVer precedence rules.
func versionLTE(a, b string) bool {
	return compareSemver(parseSemver(a), parseSemver(b)) <= 0
}
