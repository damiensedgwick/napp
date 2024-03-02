package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

	ok, err := createProject(name)
	if err != nil {
		fmt.Println("Project creation failed:", err)
		os.Exit(1)
	}

	if ok {
		fmt.Println("Project created successfully!")
		fmt.Println("Please 'cd' into your new project and run 'go mod init <insert-your-init-path> && go mod tidy'")
	}
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

func createProject(projectName string) (bool, error) {
	err := os.Mkdir(projectName, 0755)
	if err != nil {
		return false, fmt.Errorf("error creating project directory: %w", err)
	}

	subfolders := []string{"cmd", "templates", "static"}
	for _, folder := range subfolders {
		folderPath := fmt.Sprintf("%s/%s", projectName, folder)

		err := os.Mkdir(folderPath, 0755)
		if err != nil {
			return false, fmt.Errorf("error creating subfolder %s: %w", folder, err)
		}
	}

	createGoMainFile(projectName)
	createHtmlFile(projectName)
	createHtmxFile(projectName)
	createCssFile(projectName)
	createIgnoreFile(projectName)
	createDotEnvFile(projectName)
	createSqliteDbFile(projectName)

	return true, nil
}

func createGoMainFile(projectName string) {
	mainGoContent := `package main

import (
	"fmt"
	"html/template"
	"io"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/mattn/go-sqlite3"
)
	
type Template struct {
	tmpl *template.Template
}
	
func newTemplate() *Template {
	return &Template{
		tmpl: template.Must(template.ParseGlob("templates/*.html")),
	}
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.tmpl.ExecuteTemplate(w, name, data)
}

var db *sqlx.DB

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("error loading godotenv")
	}

	db = sqlx.MustConnect("sqlite3", os.Getenv("AUTH_DIARIES_DB_PATH"))

	var message string
	err = db.Ping()
	if err == nil {
		message = "Successfully connected to DB"
	}

	e := echo.New()

	e.Static("/static", "static")
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Secure())

	e.Renderer = newTemplate()

	e.GET("/", func(c echo.Context) error {
		return c.Render(200, "index", newPageData(message))
	})

	e.Logger.Fatal(e.Start(":8080"))
}

type PageData struct {
	Message string
}

func newPageData(message string) PageData {
	return PageData{
		Message: message,
	}
}
`

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
	indexHTMLContent := `{{ block "index" . }}
<!DOCTYPE html>
<html lang="en">

<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Napp | Nano App | Go, HTMX & SQLite</title>
	<meta name="description"
	content="A command line tool that helps you build and test web app ideas blazingly-fast with a streamlined Go, HTMX, and SQLite stack. Authored by Damien Sedgwick.">
	<link href="static/styles.css" rel="stylesheet">
	<script src="static/htmx.min.js"></script>
</head>

<body>
	<div>
	<h1>Hello from Napp!</h1>
	{{ if .Message }}
	<p>{{ .Message }}</p>
	{{ end }}
	</div>
</body>

</html>
{{ end }}
`

	filePath := filepath.Join(projectName, "templates", "index.html")

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

func createHtmxFile(projectName string) {
	htmxContent := `console.log('Hello, World!');`

	filePath := filepath.Join(projectName, "static", "htmx.min.js")

	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println("error creating styles.css file: ", err)
	}
	defer f.Close()

	_, err = f.WriteString(htmxContent)
	if err != nil {
		fmt.Println("error writing styles.css content to file: ", err)
	}
}

func createCssFile(projectName string) {
	cssContent := `* {
	margin: 0;
	padding: 0;
	box-sizing: border-box;
}
	
html,
body {
	height: 100%;
	width: 100%;
}
	
body {
	font-family: courier;
}
	
div {
	height: 100%;
	width: 100%;
	display: flex;
	flex-direction: column;
	justify-content: center;
	align-items: center;
}
`

	filePath := filepath.Join(projectName, "static", "styles.css")

	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println("error creating styles.css file: ", err)
	}
	defer f.Close()

	_, err = f.WriteString(cssContent)
	if err != nil {
		fmt.Println("error writing styles.css content to file: ", err)
	}
}

func createIgnoreFile(projectName string) {
	ignoreContent := `# Created by https://www.toptal.com/developers/gitignore/api/go,linux,windows,macos
# Edit at https://www.toptal.com/developers/gitignore?templates=go,linux,windows,macos

.env

### Go ###
# If you prefer the allow list template instead of the deny list, see community template:
# https://github.com/github/gitignore/blob/main/community/Golang/Go.AllowList.gitignore
#
# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with 'go test -c'
*.test

# Output of the go coverage tool, specifically when used with LiteIDE
*.out

# Dependency directories (remove the comment below to include it)
# vendor/

# Go workspace file
go.work

### Linux ###
*~

# temporary files which can be created if a process still has a handle open of a deleted file
.fuse_hidden*

# KDE directory preferences
.directory

# Linux trash folder which might appear on any partition or disk
.Trash-*

# .nfs files are created when an open file is removed but is still being accessed
.nfs*

### macOS ###
# General
.DS_Store
.AppleDouble
.LSOverride

# Icon must end with two \r
Icon


# Thumbnails
._*

# Files that might appear in the root of a volume
.DocumentRevisions-V100
.fseventsd
.Spotlight-V100
.TemporaryItems
.Trashes
.VolumeIcon.icns
.com.apple.timemachine.donotpresent

# Directories potentially created on remote AFP share
.AppleDB
.AppleDesktop
Network Trash Folder
Temporary Items
.apdisk

### macOS Patch ###
# iCloud generated files
*.icloud

### Windows ###
# Windows thumbnail cache files
Thumbs.db
Thumbs.db:encryptable
ehthumbs.db
ehthumbs_vista.db

# Dump file
*.stackdump

# Folder config file
[Dd]esktop.ini

# Recycle Bin used on file shares
$RECYCLE.BIN/

# Windows Installer files
*.cab
*.msi
*.msix
*.msm
*.msp

# Windows shortcuts
*.lnk

# End of https://www.toptal.com/developers/gitignore/api/go,linux,windows,macos
bin
auth-diaries.db
`

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
	envVarName := strings.ReplaceAll(strings.ToUpper(projectName), "-", "_")
	dbFilename := strings.ToLower(projectName) + ".db"

	dotenvContent := fmt.Sprintf(`%s_DB_PATH="%s"`, envVarName, dbFilename)

	filePath := filepath.Join(projectName, ".env")

	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println("error creating .env file: ", err)
	}
	defer f.Close()

	_, err = f.WriteString(dotenvContent)
	if err != nil {
		fmt.Println("error writing .env content to file: ", err)
	}
}

func createSqliteDbFile(projectName string) {
	dbfileName := strings.ToLower(projectName) + ".db"
	filePath := filepath.Join(projectName, dbfileName)

	_, err := os.Create(filePath)
	if err != nil {
		fmt.Println("error creating .env file: ", err)
	}
}
