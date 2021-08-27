package helpers

import (
	"math/rand"
	"time"
)

var random = rand.New(rand.NewSource(time.Now().UnixNano()))

func RandomIntBetween(low, hi int) int {
	return low + random.Intn(hi-low)
}
