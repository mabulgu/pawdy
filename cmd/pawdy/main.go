package main

import (
	"fmt"
	"os"

	"github.com/mabulgu/pawdy/internal/cli"
)

func main() {
	// Print branding on startup
	fmt.Println("ʕ•ᴥ•ʔ  hi, I'm Pawdy — your bare-metal onboarding buddy")
	
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
