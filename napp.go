package main

import (
	"fmt"
	"os"
)

type name struct {
	NotEnoughArguments string
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("No args provided!")
		os.Exit(1)
	}

	if len(args) > 1 {
		fmt.Println("Too many args provided!")
		os.Exit(1)
	}

	name := args[0]
	// TODO: check for illegal characters

	fmt.Println("Project Name: ", name)
}
