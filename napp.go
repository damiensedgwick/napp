package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/urfave/cli"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

//go:embed all:source
var source embed.FS

func main() {
	app := &cli.App{
		Name:      "napp",
		UsageText: "[command] [command options]",
		Version:   "v1.4.0",
		Description: `A command line tool that bootstraps Go, HTMX and SQLite web
	 applications and Dockerises them for ease of deployment`,
		Commands: []cli.Command{
			{
				Name:      "init",
				ShortName: "i",
				Usage:     "Initialise a new napp project ready for development",
				UsageText: "napp init <project-name>",
				Action: func(cCtx *cli.Context) error {
					if len(cCtx.Args()) != 1 {
						msg := fmt.Sprintf(
							"Oops! Received %v arguments, wanted 1",
							len(cCtx.Args()),
						)
						return cli.NewExitError(msg, 1)
					}

					projectname := cCtx.Args().Get(0)

					if isInvalidProjectName(projectname) {
						return cli.NewExitError(
							"Oops! Project name must be in the following format: <project-name>",
							1,
						)
					}

					ok, _ := createProject(projectname)
					if ok {
						fmt.Println("Successfully created " + projectname + ", next steps:")
						fmt.Println("cd " + projectname)
						fmt.Println("go mod init <path/your-project")
						fmt.Println("go mod tidy")
						fmt.Println("go run cmd/main.go")
					}

					return nil
				},
			},
		},
		Author: "Damien Sedgwick",
		Email:  "damienksedgwick@gmail.com",
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func isInvalidProjectName(name string) bool {
	pattern := "^[a-z0-9-]+$"

	matched, err := regexp.MatchString(pattern, name)
	if err != nil {
		return true
	}

	return !matched
}

func createProject(projectName string) (bool, error) {
	err := os.Mkdir(projectName, 0755)
	if err != nil {
		return false, fmt.Errorf("error creating project directory: %w", err)
	}

	subfolders := []string{"cmd", "template", "static"}
	for _, folder := range subfolders {
		folderPath := fmt.Sprintf("%s/%s", projectName, folder)

		err := os.Mkdir(folderPath, 0755)
		if err != nil {
			return false, fmt.Errorf("error creating subfolder %s: %w", folder, err)
		}
	}

	createGoMainFile(projectName)
	createHtmlFile(projectName)
	createDashboardHtmlFile(projectName)
	createHtmxFile(projectName)
	createTwColorsFile(projectName)
	createCssFile(projectName)
	createIgnoreFile(projectName)
	createDotEnvFile(projectName)
	createSqliteDbFile(projectName)
	createDockerfile(projectName)

	return true, nil
}

func createGoMainFile(projectName string) {
	sessEnv := strings.ReplaceAll(strings.ToUpper(projectName), "-", "_") + "_COOKIE_STORE_SECRET"
	dbEnv := strings.ReplaceAll(strings.ToUpper(projectName), "-", "_") + "_DB_PATH"

	mainGoTemplate, err := source.ReadFile("source/cmd/main.go")
	if err != nil {
		fmt.Println(fmt.Errorf("error reading source main.go file: %w", err))
	}

	mainGoContent := fmt.Sprintf(string(mainGoTemplate), sessEnv, dbEnv)

	filePath := filepath.Join(projectName, "cmd", "main.go")

	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println("error creating main.go file: ", err)
	}
	defer f.Close()

	_, err = f.WriteString(mainGoContent)
	if err != nil {
		fmt.Println("error writing main.go content to file: ", err)
	}
}

func createHtmlFile(projectName string) {
	pn := strings.ReplaceAll(projectName, "-", " ")

	caser := cases.Title(language.English)
	title := caser.String(pn)

	indexHTMLTemplate, err := source.ReadFile("source/template/index.html")
	if err != nil {
		fmt.Println(fmt.Errorf("error reading source index.html file: %w", err))
	}

	indexHTMLContent := fmt.Sprintf(string(indexHTMLTemplate), title, title, title, title)

	filePath := filepath.Join(projectName, "template", "index.html")

	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println("error creating index.html file: ", err)
	}
	defer f.Close()

	_, err = f.WriteString(indexHTMLContent)
	if err != nil {
		fmt.Println("error writing index.html content to file: ", err)
	}
}

func createDashboardHtmlFile(projectName string) {
	pn := strings.ReplaceAll(projectName, "-", " ")

	caser := cases.Title(language.English)
	title := caser.String(pn)

	dashboardHTMLTemplate, err := source.ReadFile("source/template/dashboard.html")
	if err != nil {
		fmt.Println(fmt.Errorf("error reading source dashboard.html file: %w", err))
	}

	dashboardHTMLContent := fmt.Sprintf(string(dashboardHTMLTemplate), title)

	filePath := filepath.Join(projectName, "template", "dashboard.html")

	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println("error creating dashboard.html file: ", err)
	}
	defer f.Close()

	_, err = f.WriteString(dashboardHTMLContent)
	if err != nil {
		fmt.Println("error writing dashboard.html content to file: ", err)
	}
}

func createHtmxFile(projectName string) {
	htmxJsContent, err := source.ReadFile("source/static/htmx.min.js")
	if err != nil {
		fmt.Println(fmt.Errorf("error reading source htmx.min.js file: %w", err))
	}

	filePath := filepath.Join(projectName, "static", "htmx.min.js")

	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println("error creating styles.css file: ", err)
	}
	defer f.Close()

	_, err = f.WriteString(string(htmxJsContent))
	if err != nil {
		fmt.Println("error writing styles.css content to file: ", err)
	}
}

func createTwColorsFile(projectName string) {
	cssContent, err := source.ReadFile("source/static/twcolors.min.css")
	if err != nil {
		fmt.Println(fmt.Errorf("error reading source htmx.min.js file: %w", err))
	}

	filePath := filepath.Join(projectName, "static", "twcolors.min.css")

	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println("error creating twcolors.min.css file: ", err)
	}
	defer f.Close()

	_, err = f.WriteString(string(cssContent))
	if err != nil {
		fmt.Println("error writing twcolors.min.css content to file: ", err)
	}
}

func createCssFile(projectName string) {
	cssContent, err := source.ReadFile("source/static/styles.css")
	if err != nil {
		fmt.Println(fmt.Errorf("error reading source htmx.min.js file: %w", err))
	}

	filePath := filepath.Join(projectName, "static", "styles.css")

	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println("error creating styles.css file: ", err)
	}
	defer f.Close()

	_, err = f.WriteString(string(cssContent))
	if err != nil {
		fmt.Println("error writing styles.css content to file: ", err)
	}
}

func createIgnoreFile(projectName string) {
	dbFilename := strings.ToLower(projectName) + ".db"
	envFilename := ".env"

	ignoreTemplate, err := source.ReadFile("source/.gitignore")
	if err != nil {
		fmt.Println(fmt.Errorf("error reading source .gitignore file: %w", err))
	}

	ignoreContent := fmt.Sprintf(
		string(ignoreTemplate),
		envFilename,
		dbFilename,
	)

	filePath := filepath.Join(projectName, ".gitignore")

	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println("error creating .gitignore file: ", err)
	}
	defer f.Close()

	_, err = f.WriteString(ignoreContent)
	if err != nil {
		fmt.Println("error writing .gitignore content to file: ", err)
	}
}

func createDotEnvFile(projectName string) {
	dbEnv := strings.ReplaceAll(strings.ToUpper(projectName), "-", "_")
	sessEnv := strings.ReplaceAll(strings.ToUpper(projectName), "-", "_")
	sessSecret := "secret"
	dbFilename := strings.ToLower(projectName) + ".db"

	dotenvTemplate, err := source.ReadFile("source/.env")
	if err != nil {
		fmt.Println(fmt.Errorf("error reading source .env file: %w", err))
	}

	dotenvContent := fmt.Sprintf(string(dotenvTemplate), dbEnv, dbFilename, sessEnv, sessSecret)

	filePath := filepath.Join(projectName, ".env")

	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println("error creating .env file: ", err)
	}
	defer f.Close()

	_, err = f.WriteString(dotenvContent)
	if err != nil {
		fmt.Println("error writing database content to file: ", err)
	}
}

func createSqliteDbFile(projectName string) {
	dbfileName := strings.ToLower(projectName) + ".db"
	filePath := filepath.Join(projectName, dbfileName)

	_, err := os.Create(filePath)
	if err != nil {
		fmt.Println("error creating database file: ", err)
	}
}

func createDockerfile(projectName string) {
	dockerfileContent, err := source.ReadFile("source/Dockerfile")
	if err != nil {
		fmt.Println(fmt.Errorf("error reading source Dockerfile file: %w", err))
	}

	filePath := filepath.Join(projectName, "Dockerfile")

	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println("error creating Dockerfile file: ", err)
	}
	defer f.Close()

	_, err = f.WriteString(string(dockerfileContent))
	if err != nil {
		fmt.Println("error writing Dockerfile content to file: ", err)
	}
}
