package internal

import (
	"log"
	"math/rand"
)

func CheckError(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func RandomIntBetween(low, hi int) int {
	return low + rand.Intn(hi-low)
}