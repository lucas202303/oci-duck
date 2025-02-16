package utils

import (
	//"log"
	"math/rand"
	"time"
)

func SleepRandomSecond(min, max int32) {
	var second int32
	if min <= 0 || max <= 0 {
		second = 1
	} else if min >= max {
		second = max
	} else {
		second = rand.Int31n(max-min) + min
	}
	//log.Printf("Sleep %d Second...\n", second)
	time.Sleep(time.Duration(second) * time.Second)
}
