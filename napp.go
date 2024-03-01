package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
)

var (
	ErrNoArguments      = errors.New("no arguments provided")
	ErrTooManyArguments = errors.New("too many arguments provided")
	ErrInvalidName      = errors.New("invalid project name")
)

func main() {
	args := os.Args[1:]

	if err := validateArgs(args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	name := args[0]

	// TODO: do the thing, create the project!

	fmt.Println("Project Name: ", name)
}

func validateArgs(args []string) error {
	if len(args) == 0 {
		return ErrNoArguments
	}

	if len(args) > 1 {
		return ErrTooManyArguments
	}

	name := args[0]
	if name == "--help" {
		fmt.Println("You can create a new Nano App with the command 'napp <project-name>'")
		os.Exit(0)
	}

	if isInvalidProjectName(name) {
		return ErrInvalidName
	}

	return nil
}

func isInvalidProjectName(name string) bool {
	pattern := "^[a-z0-9-]+$"

	matched, err := regexp.MatchString(pattern, name)
	if err != nil {
		return true
	}

	return !matched
}

