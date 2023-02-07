package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func main() {
	cmd := exec.Command("ls", "-a")
	var out strings.Builder
	cmd.Stdout = &out
	err := cmd.Run()
	output := out.String()

	fmt.Println("output: ", output)
	if err != nil {
		fmt.Println("errors: ", err)
	}

}
