package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	time.Sleep(10 * time.Second)
	random := rand.Int31()
	fmt.Println("Sleeped ", random)
}
