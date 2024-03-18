package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/urfave/cli"
)

func main() {
	app := &cli.App{
		Name:      "napp",
		UsageText: "[command] [command options]",
		Version:   "v0.3.2",
		Description: `A command line tool that bootstraps Go, HTMX and SQLite web
	 applications and Dockerises them for ease of deployment`,
		Commands: []cli.Command{
			{
				Name:      "init",
				ShortName: "i",
				Usage:     "Initialise a new napp project and start building",
				UsageText: "napp init <project-name>",
				Action: func(cCtx *cli.Context) error {
					if len(cCtx.Args()) != 1 {
						msg := fmt.Sprintf("Oops! Received %v arguments, wanted 1", len(cCtx.Args()))
						return cli.NewExitError(msg, 1)
					}

					fmt.Println("correct number of arguments... well done.")

					return nil
				},
			},
		},
		Flags:     []cli.Flag{},
		Compiled:  time.Time{},
		Authors:   []cli.Author{},
		Copyright: "",
		Author:    "Damien Sedgwick",
		Email:     "damienksedgwick@gmail.com",
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
