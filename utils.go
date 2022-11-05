package pacemaker

import (
	"crypto"
	"encoding/hex"
	"strings"

	"golang.org/x/exp/constraints"
)

func AtLeast(n int64) func(int64) int64 {
	return func(m int64) int64 {
		if m < n {
			return n
		}
		return m
	}
}

func Sha1Hash(s string) string {
	data := make([]byte, 0, 40)
	hasher := crypto.SHA1.New()
	n, err := hasher.Write([]byte(s))
	if err != nil {
		panic(err)
	}
	if n != len(s) {
		panic("insufficient bytes read")
	}
	return hex.EncodeToString(hasher.Sum(data))
}

func errIsRedisNoScript(err error) bool {
	return strings.HasPrefix(err.Error(), "NOSCRIPT")
}

func min[T constraints.Integer](a, b T) T {
	if a > b {
		return b
	}
	return a
}
