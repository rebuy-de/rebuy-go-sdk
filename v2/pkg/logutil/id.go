package logutil

import (
	"math/rand"
	"time"
)

var random = rand.New(rand.NewSource(time.Now().UnixNano()))

const idAlphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randomString(l int) string {
	var (
		b   = make([]byte, l)
		max = len(idAlphabet)
	)

	for i := range b {
		b[i] = idAlphabet[random.Intn(max)]
	}

	return string(b)
}
