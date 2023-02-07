package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	time.Sleep(7 * time.Second)
	random := rand.Int31()
	fmt.Println("Random outputs", random)
}
