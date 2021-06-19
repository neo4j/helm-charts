package internal

import (
	"math/rand"
)

func RandomIntBetween(low, hi int) int {
	return low + rand.Intn(hi-low)
}