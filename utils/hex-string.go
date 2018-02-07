package utils

import (
	"fmt"
	"math/rand"
	"time"
)

var src *rand.Rand

func init() {
	src = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func GenerateHexString() string {
	res := fmt.Sprintf("%x", src.Uint64())
	return res
}
