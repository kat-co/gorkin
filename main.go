package main

import (
	"fmt"
	"os/exec"
	"strings"
	"os"
	"flag"
	
)

const version = "0.0.1"

func main() {

	var (
		help = flag.Bool("help", false, "Get usage on gorkin.")
		initialize = flag.Bool("init", false, "Initialize a Gherkin structure.")
	)

	flag.Parse()

	if *help {
		fmt.Fprintf(os.Stderr, "gorkin v%s:\n", version)
		flag.PrintDefaults()
		return
	}

	if *initialize {
		if err := os.MkdirAll("./features/steps", 0700); err != nil {
			fmt.Fprintf(os.Stderr, "error initializing structure: %v", err)
			return
		}
		fmt.Println("Gherkin structure generated.")
		return
	}

	if _, err := os.Stat("features"); os.IsNotExist(err) {
		fmt.Printf("could not find a features directory.")
		return
	}
	
	cmd := exec.Command("go", "test", "./features/steps/...")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf(`error running "%s": %s`, strings.Join(cmd.Args, " "), string(out))
		return
	}

	fmt.Print(string(out))
}
