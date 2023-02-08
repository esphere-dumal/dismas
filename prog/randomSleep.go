package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"
)

func main() {
	if len(os.Args) == 2 {
		timeout, err := strconv.Atoi(os.Args[1])
		if err != nil {
			fmt.Println("Input is not a number")
			return
		}
		time.Sleep(time.Duration(timeout) * time.Second)
	} else {
		time.Sleep(7 * time.Second)
	}
	random := rand.Int31()
	fmt.Println("Random outputs", random)
}
