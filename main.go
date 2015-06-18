package main

import (
	"fmt"

	"os/exec"
	"strings"
	"os"
)

func main() {


	if _, err := os.Stat("features"); os.IsNotExist(err) {
		fmt.Printf("could not find a features directory.")
		return
	}
	
	cmd := exec.Command("go", "test", "-v", "./features/steps/...")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf(`error running "%s": %s`, strings.Join(cmd.Args, " "), string(out))
		return
	}

	fmt.Print(string(out))
}
