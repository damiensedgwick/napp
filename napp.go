package main

import (
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

func main() {
	app := &cli.App{
		Name:      "napp",
		UsageText: "[command] [command options]",
		Version:   "v0.6.1",
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
						fmt.Println("go mod init")
						fmt.Println("go mod tidy")
						fmt.Println("make all")
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
	createMakefile(projectName)
	createDockerfile(projectName)

	return true, nil
}

func createGoMainFile(projectName string) {
	sessEnv := strings.ReplaceAll(strings.ToUpper(projectName), "-", "_") + "_COOKIE_STORE_SECRET"
	dbEnv := strings.ReplaceAll(strings.ToUpper(projectName), "-", "_") + "_DB_PATH"

	mainGoContent := fmt.Sprintf(`package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/mail"
	"os"
	"time"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Template struct {
	tmpl *template.Template
}

func newTemplate() *Template {
	return &Template{
		tmpl: template.Must(template.ParseGlob("template/*.html")),
	}
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.tmpl.ExecuteTemplate(w, name, data)
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("error loading godotenv")
	}

	e := echo.New()
	e.Renderer = newTemplate()
	e.Static("/static", "static")
	e.Use(middleware.Recover())
	e.Use(middleware.Secure())
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}\n",
	}))
	store := sessions.NewCookieStore([]byte(os.Getenv("%s")))
	e.Use(session.Middleware(store))

	db, err := gorm.Open(sqlite.Open(os.Getenv("%s")), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&Lead{}, &User{})

	e.GET("/", homepageHandler())
	e.POST("/join-waitlist", joinWaitlistHandler(db))
	e.GET("/auth/sign-in", signIn())
	e.POST("/auth/sign-in", signInWithEmailAndPassword(db))
	e.GET("/auth/sign-up", signUp())
	e.POST("/auth/sign-up", signUpWithEmailAndPassword(db))
	e.POST("/auth/sign-out", signOut())
	e.GET("/dashboard", dashboardHandler())

	e.Logger.Fatal(e.Start(":8080"))
}

type PageData struct {
	User     User
	LeadForm FormData
}

func newPageData(user User, leadForm FormData) PageData {
	return PageData{
		User:     user,
		LeadForm: leadForm,
	}
}

func homepageHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, _ := session.Get("session", c)
		if sess.Values["user"] != nil {
			var user User
			err := json.Unmarshal(sess.Values["user"].([]byte), &user)
			if err != nil {
				fmt.Println("error unmarshalling user value")
				return err
			}

			return c.Render(200, "index", newPageData(user, newFormData()))
		}

		return c.Render(200, "index", nil)
	}
}

type Lead struct {
	gorm.Model
	Email     string
	CreatedAt time.Time
	UpdatedAt *time.Time
}

type FormData struct {
	Errors map[string]string
	Values map[string]string
}

func newFormData() FormData {
	return FormData{
		Errors: map[string]string{},
		Values: map[string]string{},
	}
}

func joinWaitlistHandler(db *gorm.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		email := c.FormValue("email")
		_, err := mail.ParseAddress(email)
		if err != nil {
			return c.Render(422, "waitlist", FormData{
				Errors: map[string]string{
					"email": "Oops! That email address appears to be invalid",
				},
				Values: map[string]string{
					"email": email,
				},
			})
		}

		if leadExists(email, db) {
			return c.Render(422, "waitlist", FormData{
				Errors: map[string]string{
					"email": "Oops! It appears you are already subscribed",
				},
				Values: map[string]string{
					"email": email,
				},
			})
		}

		lead := Lead{
			Email: email,
		}

		if err := db.Create(&lead).Error; err != nil {
			return c.Render(500, "waitlist", FormData{
				Errors: map[string]string{
					"email": "Oops! It appears we have had an error",
				},
				Values: map[string]string{},
			})
		}

		return c.Render(200, "waitlist-joined", nil)
	}
}

func leadExists(email string, db *gorm.DB) bool {
	var lead Lead
	err := db.First(&lead, "email = ?", email).Error

	return err != gorm.ErrRecordNotFound
}

func userExists(email string, db *gorm.DB) bool {
	var user User
	err := db.First(&user, "email = ?", email).Error

	return err != gorm.ErrRecordNotFound
}

type User struct {
	gorm.Model
	Name      string
	Email     string
	Password  string
	Role      string
	CreatedAt time.Time
	UpdatedAt *time.Time
}

func newUser(name string, email string, password string, role string, created_at time.Time, updated_at *time.Time) User {
	return User{
		Name:      name,
		Email:     email,
		Password:  password,
		Role:      role,
		CreatedAt: created_at,
		UpdatedAt: updated_at,
	}
}

func signUp() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Render(200, "sign-up-form", nil)
	}
}

func signUpWithEmailAndPassword(db *gorm.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		name := c.FormValue("name")
		email := c.FormValue("email")
		password := c.FormValue("password")

		_, err := mail.ParseAddress(email)
		if err != nil {
			return c.Render(422, "sign-up-form", FormData{
				Errors: map[string]string{
					"email": "Oops! That email address appears to be invalid",
				},
				Values: map[string]string{
					"email": email,
				},
			})
		}

		if userExists(email, db) {
			return c.Render(422, "sign-up-form", FormData{
				Errors: map[string]string{
					"email": "Oops! It appears you are already registered",
				},
				Values: map[string]string{
					"email": email,
				},
			})
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
		if err != nil {
			log.Fatal("Could not hash sign up password")
		}

		// Check if this is the first user
		var count int64
		if err := db.Model(&User{}).Count(&count).Error; err != nil {
			return c.Render(500, "sign-up-form", FormData{
				Errors: map[string]string{
					"general": "Oops! It appears we have had an error",
				},
				Values: map[string]string{},
			})
		}

		role := "user"
		if count == 0 {
			role = "admin"
		}

		user := User{
			Name:      name,
			Email:     email,
			Password:  string(hash),
			Role:      role,
			CreatedAt: time.Now(),
		}

		if err := db.Create(&user).Error; err != nil {
			return c.Render(500, "sign-up-form", FormData{
				Errors: map[string]string{
					"email": "Oops! It appears we have had an error",
				},
				Values: map[string]string{},
			})
		}

		return c.Render(200, "index", nil)
	}
}

func signIn() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Render(200, "sign-in-form", nil)
	}
}

func signInWithEmailAndPassword(db *gorm.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		email := c.FormValue("email")
		password := c.FormValue("password")

		_, err := mail.ParseAddress(email)
		if err != nil {
			return c.Render(422, "sign-in-form", FormData{
				Errors: map[string]string{
					"email": "Oops! That email address appears to be invalid",
				},
				Values: map[string]string{
					"email": email,
				},
			})
		}

		var user User
		db.First(&user, "email = ?", email)
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
			return c.Render(422, "sign-in-form", FormData{
				Errors: map[string]string{
					"email": "Oops! Email address or password is incorrect.",
				},
				Values: map[string]string{
					"email": email,
				},
			})
		}

		sess, _ := session.Get("session", c)
		sess.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   86400 * 7,
			HttpOnly: true,
		}

		userBytes, err := json.Marshal(user)
		if err != nil {
			fmt.Println("error marshalling user value")
			return err
		}

		sess.Values["user"] = userBytes

		err = sess.Save(c.Request(), c.Response())
		if err != nil {
			fmt.Println("error saving session: ", err)
			return err
		}

		return c.Render(200, "dashboard", newDashboardData(user))
	}
}

func signOut() echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, _ := session.Get("session", c)
		sess.Options.MaxAge = -1
		err := sess.Save(c.Request(), c.Response())
		if err != nil {
			fmt.Println("error saving session")
			return err
		}

		return c.Render(200, "index", nil)
	}
}

type DashboardData struct {
	User User
}

func newDashboardData(user User) DashboardData {
	return DashboardData{
		User: user,
	}
}

func dashboardHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, _ := session.Get("session", c)
		if sess.Values["user"] != nil {
			var user User
			err := json.Unmarshal(sess.Values["user"].([]byte), &user)
			if err != nil {
				fmt.Println("error unmarshalling user value")
				return err
			}

			return c.Render(200, "dashboard", newDashboardData(user))
		}

		return c.Redirect(http.StatusFound, "/")
	}
}

`, sessEnv, dbEnv)

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

	indexHTMLContent := fmt.Sprintf(`{{ block "index" . }}
<!DOCTYPE html>

<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Napp | Nano App | Go, HTMX & SQLite</title>
  <meta name="description"
    content="A command line tool that helps you build and test web app ideas blazingly-fast with a streamlined Go, HTMX, and SQLite stack. Authored by Damien Sedgwick.">
  <link href="static/twcolors.min.css" rel="stylesheet">
  <link href="static/styles.css" rel="stylesheet">
  <script src="static/htmx.min.js"></script>
</head>

<body id="body">
  <nav class="nav">
    <div class="container">
      <div class="nav__content">
	    <a class="nav__brand" href="/" title="Heating Oil Tracker Home">
	      %s
	    </a>
	    <ul class="nav__list">
	      {{ if not .User }}
	      <li class="nav__item">
		    <button class="nav__link" hx-get="/auth/sign-in" hx-target="body">Sign In</button>
	      </li>
	      {{ end }}

     	  {{ if .User }}
	      <li class="nav__item">
		    <a class="nav__link" href="/dashboard" title="Dashboard">Dashboard</a>
	      </li>
	      <li class="nav__item">
		    <button class="nav__link" hx-post="/auth/sign-out" hx-target="body">Sign Out</button>
	      </li>
          {{ end }}
	    </ul>
      </div>
    </div>
  </nav>
  <main>
    <div class="hero">
      <h1 class="hero__title">%s</h1>
      <p class="hero__intro">Join our waiting list and you'll be the first to know when we launch, ensuring you don't miss out on any exciting updates or early access opportunities.</p>
      {{ template "waitlist" .LeadForm }}
    </div>
  </main>

  <script type="text/javascript">
  document.addEventListener("DOMContentLoaded", (event) => {
    document.body.addEventListener('htmx:beforeSwap', function (evt) {
      if (evt.detail.xhr.status === 422 || evt.detail.xhr.status === 500) {
        console.log("setting status to paint");
        // allow 422 responses to swap as we are using this as a signal that
        // a form was submitted with bad data and want to rerender with the
        // errors
        //
        // set isError to false to avoid error logging in console
        evt.detail.shouldSwap = true;
        evt.detail.isError = false;
      }
    });
  });
  </script>
</body>
</html>
{{ end }}

{{ block "waitlist" . }}      
<form class="waitlist-form" id="waitlist-form" hx-post="/join-waitlist" hx-swap="outerHTML">
  <div class="waitlist-form__group">
    <label class="waitlist-form__label" for="email">
      <input 
        class="waitlist-form__input"
        type="text"
        name="email"
        placeholder="Please enter your email"
        {{ if .Values.email}}
        value="{{ .Values.email }}"
        {{end}} 
        required
      >
    </label>

    <button class="btn waitlist-form__btn" type="submit">Join Waitlist</button>
  </div>

  {{ if .Errors.email }}
  <p class="waitlist-form__message waitlist-form__message-error">
    {{ .Errors.email }}
  </p>
  {{ end }}
</form>
{{ end }}

{{ block "waitlist-joined" . }}
<p>Thanks! You successfully joined our waitlist</p>
{{ end }}

{{ block "sign-up-form" . }}
<div class="auth-form__wrapper">
  <form class="auth-form" id="sign-up-form" hx-post="/auth/sign-up" hx-target="body">
    <p class="auth-form__title">
	  %s
    </p>

    <div class="auth-form__group">
      <label class="auth-form__label" for="name">
        Name
      </label>
      <input id="name" class="auth-form__input" type="text" name="name" autocomplete="name" value="" required>
    </div>

    <div class="auth-form__group">
      <label class="auth-form__label" for="email">
        Email
      </label>
      <input id="email" class="auth-form__input" type="text" name="email" autocomplete="email" value="" required>
    </div>

    <div class="auth-form__group">
      <label class="auth-form__label" for="password">
        Password
      </label>
      <input id="password" class="auth-form__input" type="password" name="password" value="" required>
    </div>

    <button class="btn auth-form__btn" type="submit">Sign In</button>

    {{ if .Errors.email}}
    <p class="auth-form__message auth-form__message-error">
      {{ .Errors.email}}
    </p>
    {{ end }}

    <p class="auth-form__type">Already have an account? <button class="btn btn-ghost" type="button"
        hx-get="/auth/sign-in" hx-target="body">Sign In</button></p>
  </form>
</div>
{{ end }}

{{ block "sign-in-form" . }}
<div class="auth-form__wrapper">
  <form class="auth-form" id="sign-in-form" hx-post="/auth/sign-in" hx-target="body">
    <p class="auth-form__title">
      %s
    </p>
    <div class="auth-form__group">
      <label class="auth-form__label" for="email">
        Email
      </label>
      <input id="email" class="auth-form__input" type="text" name="email" autocomplete="email" value="" required>
    </div>

    <div class="auth-form__group">
      <label class="auth-form__label" for="password">
        Password
      </label>
      <input id="password" class="auth-form__input" type="password" name="password" value="" required>
    </div>

    <button class="btn auth-form__btn" type="submit">Sign In</button>

    {{ if .Errors.email}}
    <p class="auth-form__message auth-form__message-error">
      {{ .Errors.email}}
    </p>
    {{ end }}

    <p class="auth-form__type">Do you need an account? <button class="btn btn-ghost" type="button"
        hx-get="/auth/sign-up" hx-target="body">Register Now</button></p>
  </form>
</div>
{{ end }}

`, title, title, title, title)

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

	dashboardHTMLContent := fmt.Sprintf(`{{ block "dashboard" . }}
<!DOCTYPE html>

<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Heating Oil Tracker | Monitor and gauge your heating oil levels</title>
  <meta name="description"
    content="Effortlessly track your heating oil levels with our intuitive Heating Oil Tracker app. Stay in control of your home's warmth, ensuring you're never left in the cold. Sign up now for peace of mind!">
  <link rel="icon" type="image/x-icon" href="static/favicon.png">
  <link href="static/twcolors.min.css" rel="stylesheet">
  <link href="static/styles.css" rel="stylesheet">
  <script src="static/htmx.min.js"></script>
</head>

<body id="body">
  <div class="dashboard__wrapper">
    <aside class="dashboard__navigation">
      <div>
        <div class="dashboard__branding">
          %s
        </div>
        <ul class="dashboard__navigation-list">
          <li class="dashboard__navigation-item">
            <button class="dashboard__navigation-link">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5"
                stroke="currentColor" class="size-6">
                <path stroke-linecap="round" stroke-linejoin="round"
                  d="m2.25 12 8.954-8.955c.44-.439 1.152-.439 1.591 0L21.75 12M4.5 9.75v10.125c0 .621.504 1.125 1.125 1.125H9.75v-4.875c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125V21h4.125c.621 0 1.125-.504 1.125-1.125V9.75M8.25 21h8.25" />
              </svg>
              Dashboard
            </button>
          </li>
          <li class="dashboard__navigation-item">
            <button class="dashboard__navigation-link">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5"
                stroke="currentColor" class="size-6">
                <path stroke-linecap="round" stroke-linejoin="round"
                  d="M18 18.72a9.094 9.094 0 0 0 3.741-.479 3 3 0 0 0-4.682-2.72m.94 3.198.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0 1 12 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 0 1 6 18.719m12 0a5.971 5.971 0 0 0-.941-3.197m0 0A5.995 5.995 0 0 0 12 12.75a5.995 5.995 0 0 0-5.058 2.772m0 0a3 3 0 0 0-4.681 2.72 8.986 8.986 0 0 0 3.74.477m.94-3.197a5.971 5.971 0 0 0-.94 3.197M15 6.75a3 3 0 1 1-6 0 3 3 0 0 1 6 0Zm6 3a2.25 2.25 0 1 1-4.5 0 2.25 2.25 0 0 1 4.5 0Zm-13.5 0a2.25 2.25 0 1 1-4.5 0 2.25 2.25 0 0 1 4.5 0Z" />
              </svg>
              Team
            </button>
          </li>
          <li class="dashboard__navigation-item">
            <button class="dashboard__navigation-link">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5"
                stroke="currentColor" class="size-6">
                <path stroke-linecap="round" stroke-linejoin="round"
                  d="M2.25 12.75V12A2.25 2.25 0 0 1 4.5 9.75h15A2.25 2.25 0 0 1 21.75 12v.75m-8.69-6.44-2.12-2.12a1.5 1.5 0 0 0-1.061-.44H4.5A2.25 2.25 0 0 0 2.25 6v12a2.25 2.25 0 0 0 2.25 2.25h15A2.25 2.25 0 0 0 21.75 18V9a2.25 2.25 0 0 0-2.25-2.25h-5.379a1.5 1.5 0 0 1-1.06-.44Z" />
              </svg>
              Projects
            </button>
          </li>
          <li class="dashboard__navigation-item">
            <button class="dashboard__navigation-link">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5"
                stroke="currentColor" class="size-6">
                <path stroke-linecap="round" stroke-linejoin="round"
                  d="M6.75 3v2.25M17.25 3v2.25M3 18.75V7.5a2.25 2.25 0 0 1 2.25-2.25h13.5A2.25 2.25 0 0 1 21 7.5v11.25m-18 0A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75m-18 0v-7.5A2.25 2.25 0 0 1 5.25 9h13.5A2.25 2.25 0 0 1 21 11.25v7.5" />
              </svg>
              Calendar
            </button>
          </li>
        </ul>

        {{ if and .User (eq .User.Role "admin") }}
        <div class="dashboard__navigation-admin-separator"></div>
        <ul class="dashboard__navigation-admin-list">
          <li class="dashboard__navigation-item">
            <button class="dashboard__navigation-link">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5"
                stroke="currentColor" class="size-6">
                <path stroke-linecap="round" stroke-linejoin="round"
                  d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 0 0 2.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 0 0-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 0 0 .75-.75 2.25 2.25 0 0 0-.1-.664m-5.8 0A2.251 2.251 0 0 1 13.5 2.25H15c1.012 0 1.867.668 2.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25ZM6.75 12h.008v.008H6.75V12Zm0 3h.008v.008H6.75V15Zm0 3h.008v.008H6.75V18Z" />
              </svg>

              Leads
            </button>
          </li>
        </ul>
        {{ end }}
      </div>

      <button class="btn dashboard__navigation-sign-out" hx-post="/auth/sign-out" hx-target="body">Sign Out</button>
    </aside>
    <main class="dashboard__content">
      <p>Dashboard</p>
    </main>
  </div>

  <script type="text/javascript">
    document.addEventListener("DOMContentLoaded", (event) => {
      document.body.addEventListener('htmx:beforeSwap', function (evt) {
        if (evt.detail.xhr.status === 422) {
          console.log("setting status to paint");
          // allow 422 responses to swap as we are using this as a signal that
          // a form was submitted with bad data and want to rerender with the
          // errors
          //
          // set isError to false to avoid error logging in console
          evt.detail.shouldSwap = true;
          evt.detail.isError = false;
        }
      });
    });
  </script>
</body>

</html>
{{ end }}

`, title)

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
	htmxContent := `(function (e, t) { if (typeof define === "function" && define.amd) { define([], t) } else if (typeof module === "object" && module.exports) { module.exports = t() } else { e.htmx = e.htmx || t() } })(typeof self !== "undefined" ? self : this, function () { return function () { "use strict"; var Q = { onLoad: F, process: zt, on: de, off: ge, trigger: ce, ajax: Nr, find: C, findAll: f, closest: v, values: function (e, t) { var r = dr(e, t || "post"); return r.values }, remove: _, addClass: z, removeClass: n, toggleClass: $, takeClass: W, defineExtension: Ur, removeExtension: Br, logAll: V, logNone: j, logger: null, config: { historyEnabled: true, historyCacheSize: 10, refreshOnHistoryMiss: false, defaultSwapStyle: "innerHTML", defaultSwapDelay: 0, defaultSettleDelay: 20, includeIndicatorStyles: true, indicatorClass: "htmx-indicator", requestClass: "htmx-request", addedClass: "htmx-added", settlingClass: "htmx-settling", swappingClass: "htmx-swapping", allowEval: true, allowScriptTags: true, inlineScriptNonce: "", attributesToSettle: ["class", "style", "width", "height"], withCredentials: false, timeout: 0, wsReconnectDelay: "full-jitter", wsBinaryType: "blob", disableSelector: "[hx-disable], [data-hx-disable]", useTemplateFragments: false, scrollBehavior: "smooth", defaultFocusScroll: false, getCacheBusterParam: false, globalViewTransitions: false, methodsThatUseUrlParams: ["get"], selfRequestsOnly: false, ignoreTitle: false, scrollIntoViewOnBoost: true, triggerSpecsCache: null }, parseInterval: d, _: t, createEventSource: function (e) { return new EventSource(e, { withCredentials: true }) }, createWebSocket: function (e) { var t = new WebSocket(e, []); t.binaryType = Q.config.wsBinaryType; return t }, version: "1.9.10" }; var r = { addTriggerHandler: Lt, bodyContains: se, canAccessLocalStorage: U, findThisElement: xe, filterValues: yr, hasAttribute: o, getAttributeValue: te, getClosestAttributeValue: ne, getClosestMatch: c, getExpressionVars: Hr, getHeaders: xr, getInputValues: dr, getInternalData: ae, getSwapSpecification: wr, getTriggerSpecs: it, getTarget: ye, makeFragment: l, mergeObjects: le, makeSettleInfo: T, oobSwap: Ee, querySelectorExt: ue, selectAndSwap: je, settleImmediately: nr, shouldCancel: ut, triggerEvent: ce, triggerErrorEvent: fe, withExtensions: R }; var w = ["get", "post", "put", "delete", "patch"]; var i = w.map(function (e) { return "[hx-" + e + "], [data-hx-" + e + "]" }).join(", "); var S = e("head"), q = e("title"), H = e("svg", true); function e(e, t = false) { return new RegExp(` + "`<${e}(\\s[^>]*>|>)([\\s\\S]*?)<\\/${e}>`" + `, t ? "gim" : "im") } function d(e) { if (e == undefined) { return undefined } let t = NaN; if (e.slice(-2) == "ms") { t = parseFloat(e.slice(0, -2)) } else if (e.slice(-1) == "s") { t = parseFloat(e.slice(0, -1)) * 1e3 } else if (e.slice(-1) == "m") { t = parseFloat(e.slice(0, -1)) * 1e3 * 60 } else { t = parseFloat(e) } return isNaN(t) ? undefined : t } function ee(e, t) { return e.getAttribute && e.getAttribute(t) } function o(e, t) { return e.hasAttribute && (e.hasAttribute(t) || e.hasAttribute("data-" + t)) } function te(e, t) { return ee(e, t) || ee(e, "data-" + t) } function u(e) { return e.parentElement } function re() { return document } function c(e, t) { while (e && !t(e)) { e = u(e) } return e ? e : null } function L(e, t, r) { var n = te(t, r); var i = te(t, "hx-disinherit"); if (e !== t && i && (i === "*" || i.split(" ").indexOf(r) >= 0)) { return "unset" } else { return n } } function ne(t, r) { var n = null; c(t, function (e) { return n = L(t, e, r) }); if (n !== "unset") { return n } } function h(e, t) { var r = e.matches || e.matchesSelector || e.msMatchesSelector || e.mozMatchesSelector || e.webkitMatchesSelector || e.oMatchesSelector; return r && r.call(e, t) } function A(e) { var t = /<([a-z][^\/\0>\x20\t\r\n\f]*)/i; var r = t.exec(e); if (r) { return r[1].toLowerCase() } else { return "" } } function a(e, t) { var r = new DOMParser; var n = r.parseFromString(e, "text/html"); var i = n.body; while (t > 0) { t--; i = i.firstChild } if (i == null) { i = re().createDocumentFragment() } return i } function N(e) { return /<body/.test(e) } function l(e) { var t = !N(e); var r = A(e); var n = e; if (r === "head") { n = n.replace(S, "") } if (Q.config.useTemplateFragments && t) { var i = a("<body><template>" + n + "</template></body>", 0); return i.querySelector("template").content } switch (r) { case "thead": case "tbody": case "tfoot": case "colgroup": case "caption": return a("<table>" + n + "</table>", 1); case "col": return a("<table><colgroup>" + n + "</colgroup></table>", 2); case "tr": return a("<table><tbody>" + n + "</tbody></table>", 2); case "td": case "th": return a("<table><tbody><tr>" + n + "</tr></tbody></table>", 3); case "script": case "style": return a("<div>" + n + "</div>", 1); default: return a(n, 0) } } function ie(e) { if (e) { e() } } function I(e, t) { return Object.prototype.toString.call(e) === "[object " + t + "]" } function k(e) { return I(e, "Function") } function P(e) { return I(e, "Object") } function ae(e) { var t = "htmx-internal-data"; var r = e[t]; if (!r) { r = e[t] = {} } return r } function M(e) { var t = []; if (e) { for (var r = 0; r < e.length; r++) { t.push(e[r]) } } return t } function oe(e, t) { if (e) { for (var r = 0; r < e.length; r++) { t(e[r]) } } } function X(e) { var t = e.getBoundingClientRect(); var r = t.top; var n = t.bottom; return r < window.innerHeight && n >= 0 } function se(e) { if (e.getRootNode && e.getRootNode() instanceof window.ShadowRoot) { return re().body.contains(e.getRootNode().host) } else { return re().body.contains(e) } } function D(e) { return e.trim().split(/\s+/) } function le(e, t) { for (var r in t) { if (t.hasOwnProperty(r)) { e[r] = t[r] } } return e } function E(e) { try { return JSON.parse(e) } catch (e) { b(e); return null } } function U() { var e = "htmx:localStorageTest"; try { localStorage.setItem(e, e); localStorage.removeItem(e); return true } catch (e) { return false } } function B(t) { try { var e = new URL(t); if (e) { t = e.pathname + e.search } if (!/^\/$/.test(t)) { t = t.replace(/\/+$/, "") } return t } catch (e) { return t } } function t(e) { return Tr(re().body, function () { return eval(e) }) } function F(t) { var e = Q.on("htmx:load", function (e) { t(e.detail.elt) }); return e } function V() { Q.logger = function (e, t, r) { if (console) { console.log(t, e, r) } } } function j() { Q.logger = null } function C(e, t) { if (t) { return e.querySelector(t) } else { return C(re(), e) } } function f(e, t) { if (t) { return e.querySelectorAll(t) } else { return f(re(), e) } } function _(e, t) { e = g(e); if (t) { setTimeout(function () { _(e); e = null }, t) } else { e.parentElement.removeChild(e) } } function z(e, t, r) { e = g(e); if (r) { setTimeout(function () { z(e, t); e = null }, r) } else { e.classList && e.classList.add(t) } } function n(e, t, r) { e = g(e); if (r) { setTimeout(function () { n(e, t); e = null }, r) } else { if (e.classList) { e.classList.remove(t); if (e.classList.length === 0) { e.removeAttribute("class") } } } } function $(e, t) { e = g(e); e.classList.toggle(t) } function W(e, t) { e = g(e); oe(e.parentElement.children, function (e) { n(e, t) }); z(e, t) } function v(e, t) { e = g(e); if (e.closest) { return e.closest(t) } else { do { if (e == null || h(e, t)) { return e } } while (e = e && u(e)); return null } } function s(e, t) { return e.substring(0, t.length) === t } function G(e, t) { return e.substring(e.length - t.length) === t } function J(e) { var t = e.trim(); if (s(t, "<") && G(t, "/>")) { return t.substring(1, t.length - 2) } else { return t } } function Z(e, t) { if (t.indexOf("closest ") === 0) { return [v(e, J(t.substr(8)))] } else if (t.indexOf("find ") === 0) { return [C(e, J(t.substr(5)))] } else if (t === "next") { return [e.nextElementSibling] } else if (t.indexOf("next ") === 0) { return [K(e, J(t.substr(5)))] } else if (t === "previous") { return [e.previousElementSibling] } else if (t.indexOf("previous ") === 0) { return [Y(e, J(t.substr(9)))] } else if (t === "document") { return [document] } else if (t === "window") { return [window] } else if (t === "body") { return [document.body] } else { return re().querySelectorAll(J(t)) } } var K = function (e, t) { var r = re().querySelectorAll(t); for (var n = 0; n < r.length; n++) { var i = r[n]; if (i.compareDocumentPosition(e) === Node.DOCUMENT_POSITION_PRECEDING) { return i } } }; var Y = function (e, t) { var r = re().querySelectorAll(t); for (var n = r.length - 1; n >= 0; n--) { var i = r[n]; if (i.compareDocumentPosition(e) === Node.DOCUMENT_POSITION_FOLLOWING) { return i } } }; function ue(e, t) { if (t) { return Z(e, t)[0] } else { return Z(re().body, e)[0] } } function g(e) { if (I(e, "String")) { return C(e) } else { return e } } function ve(e, t, r) { if (k(t)) { return { target: re().body, event: e, listener: t } } else { return { target: g(e), event: t, listener: r } } } function de(t, r, n) { jr(function () { var e = ve(t, r, n); e.target.addEventListener(e.event, e.listener) }); var e = k(r); return e ? r : n } function ge(t, r, n) { jr(function () { var e = ve(t, r, n); e.target.removeEventListener(e.event, e.listener) }); return k(r) ? r : n } var me = re().createElement("output"); function pe(e, t) { var r = ne(e, t); if (r) { if (r === "this") { return [xe(e, t)] } else { var n = Z(e, r); if (n.length === 0) { b('The selector "' + r + '" on ' + t + " returned no matches!"); return [me] } else { return n } } } } function xe(e, t) { return c(e, function (e) { return te(e, t) != null }) } function ye(e) { var t = ne(e, "hx-target"); if (t) { if (t === "this") { return xe(e, "hx-target") } else { return ue(e, t) } } else { var r = ae(e); if (r.boosted) { return re().body } else { return e } } } function be(e) { var t = Q.config.attributesToSettle; for (var r = 0; r < t.length; r++) { if (e === t[r]) { return true } } return false } function we(t, r) { oe(t.attributes, function (e) { if (!r.hasAttribute(e.name) && be(e.name)) { t.removeAttribute(e.name) } }); oe(r.attributes, function (e) { if (be(e.name)) { t.setAttribute(e.name, e.value) } }) } function Se(e, t) { var r = Fr(t); for (var n = 0; n < r.length; n++) { var i = r[n]; try { if (i.isInlineSwap(e)) { return true } } catch (e) { b(e) } } return e === "outerHTML" } function Ee(e, i, a) { var t = "#" + ee(i, "id"); var o = "outerHTML"; if (e === "true") { } else if (e.indexOf(":") > 0) { o = e.substr(0, e.indexOf(":")); t = e.substr(e.indexOf(":") + 1, e.length) } else { o = e } var r = re().querySelectorAll(t); if (r) { oe(r, function (e) { var t; var r = i.cloneNode(true); t = re().createDocumentFragment(); t.appendChild(r); if (!Se(o, e)) { t = r } var n = { shouldSwap: true, target: e, fragment: t }; if (!ce(e, "htmx:oobBeforeSwap", n)) return; e = n.target; if (n["shouldSwap"]) { Fe(o, e, e, t, a) } oe(a.elts, function (e) { ce(e, "htmx:oobAfterSwap", n) }) }); i.parentNode.removeChild(i) } else { i.parentNode.removeChild(i); fe(re().body, "htmx:oobErrorNoTarget", { content: i }) } return e } function Ce(e, t, r) { var n = ne(e, "hx-select-oob"); if (n) { var i = n.split(","); for (var a = 0; a < i.length; a++) { var o = i[a].split(":", 2); var s = o[0].trim(); if (s.indexOf("#") === 0) { s = s.substring(1) } var l = o[1] || "true"; var u = t.querySelector("#" + s); if (u) { Ee(l, u, r) } } } oe(f(t, "[hx-swap-oob], [data-hx-swap-oob]"), function (e) { var t = te(e, "hx-swap-oob"); if (t != null) { Ee(t, e, r) } }) } function Re(e) { oe(f(e, "[hx-preserve], [data-hx-preserve]"), function (e) { var t = te(e, "id"); var r = re().getElementById(t); if (r != null) { e.parentNode.replaceChild(r, e) } }) } function Te(o, e, s) { oe(e.querySelectorAll("[id]"), function (e) { var t = ee(e, "id"); if (t && t.length > 0) { var r = t.replace("'", "\\'"); var n = e.tagName.replace(":", "\\:"); var i = o.querySelector(n + "[id='" + r + "']"); if (i && i !== o) { var a = e.cloneNode(); we(e, i); s.tasks.push(function () { we(e, a) }) } } }) } function Oe(e) { return function () { n(e, Q.config.addedClass); zt(e); Nt(e); qe(e); ce(e, "htmx:load") } } function qe(e) { var t = "[autofocus]"; var r = h(e, t) ? e : e.querySelector(t); if (r != null) { r.focus() } } function m(e, t, r, n) { Te(e, r, n); while (r.childNodes.length > 0) { var i = r.firstChild; z(i, Q.config.addedClass); e.insertBefore(i, t); if (i.nodeType !== Node.TEXT_NODE && i.nodeType !== Node.COMMENT_NODE) { n.tasks.push(Oe(i)) } } } function He(e, t) { var r = 0; while (r < e.length) { t = (t << 5) - t + e.charCodeAt(r++) | 0 } return t } function Le(e) { var t = 0; if (e.attributes) { for (var r = 0; r < e.attributes.length; r++) { var n = e.attributes[r]; if (n.value) { t = He(n.name, t); t = He(n.value, t) } } } return t } function Ae(e) { var t = ae(e); if (t.onHandlers) { for (var r = 0; r < t.onHandlers.length; r++) { const n = t.onHandlers[r]; e.removeEventListener(n.event, n.listener) } delete t.onHandlers } } function Ne(e) { var t = ae(e); if (t.timeout) { clearTimeout(t.timeout) } if (t.webSocket) { t.webSocket.close() } if (t.sseEventSource) { t.sseEventSource.close() } if (t.listenerInfos) { oe(t.listenerInfos, function (e) { if (e.on) { e.on.removeEventListener(e.trigger, e.listener) } }) } Ae(e); oe(Object.keys(t), function (e) { delete t[e] }) } function p(e) { ce(e, "htmx:beforeCleanupElement"); Ne(e); if (e.children) { oe(e.children, function (e) { p(e) }) } } function Ie(t, e, r) { if (t.tagName === "BODY") { return Ue(t, e, r) } else { var n; var i = t.previousSibling; m(u(t), t, e, r); if (i == null) { n = u(t).firstChild } else { n = i.nextSibling } r.elts = r.elts.filter(function (e) { return e != t }); while (n && n !== t) { if (n.nodeType === Node.ELEMENT_NODE) { r.elts.push(n) } n = n.nextElementSibling } p(t); u(t).removeChild(t) } } function ke(e, t, r) { return m(e, e.firstChild, t, r) } function Pe(e, t, r) { return m(u(e), e, t, r) } function Me(e, t, r) { return m(e, null, t, r) } function Xe(e, t, r) { return m(u(e), e.nextSibling, t, r) } function De(e, t, r) { p(e); return u(e).removeChild(e) } function Ue(e, t, r) { var n = e.firstChild; m(e, n, t, r); if (n) { while (n.nextSibling) { p(n.nextSibling); e.removeChild(n.nextSibling) } p(n); e.removeChild(n) } } function Be(e, t, r) { var n = r || ne(e, "hx-select"); if (n) { var i = re().createDocumentFragment(); oe(t.querySelectorAll(n), function (e) { i.appendChild(e) }); t = i } return t } function Fe(e, t, r, n, i) { switch (e) { case "none": return; case "outerHTML": Ie(r, n, i); return; case "afterbegin": ke(r, n, i); return; case "beforebegin": Pe(r, n, i); return; case "beforeend": Me(r, n, i); return; case "afterend": Xe(r, n, i); return; case "delete": De(r, n, i); return; default: var a = Fr(t); for (var o = 0; o < a.length; o++) { var s = a[o]; try { var l = s.handleSwap(e, r, n, i); if (l) { if (typeof l.length !== "undefined") { for (var u = 0; u < l.length; u++) { var f = l[u]; if (f.nodeType !== Node.TEXT_NODE && f.nodeType !== Node.COMMENT_NODE) { i.tasks.push(Oe(f)) } } } return } } catch (e) { b(e) } } if (e === "innerHTML") { Ue(r, n, i) } else { Fe(Q.config.defaultSwapStyle, t, r, n, i) } } } function Ve(e) { if (e.indexOf("<title") > -1) { var t = e.replace(H, ""); var r = t.match(q); if (r) { return r[2] } } } function je(e, t, r, n, i, a) { i.title = Ve(n); var o = l(n); if (o) { Ce(r, o, i); o = Be(r, o, a); Re(o); return Fe(e, r, t, o, i) } } function _e(e, t, r) { var n = e.getResponseHeader(t); if (n.indexOf("{") === 0) { var i = E(n); for (var a in i) { if (i.hasOwnProperty(a)) { var o = i[a]; if (!P(o)) { o = { value: o } } ce(r, a, o) } } } else { var s = n.split(","); for (var l = 0; l < s.length; l++) { ce(r, s[l].trim(), []) } } } var ze = /\s/; var x = /[\s,]/; var $e = /[_$a-zA-Z]/; var We = /[_$a-zA-Z0-9]/; var Ge = ['"', "'", "/"]; var Je = /[^\s]/; var Ze = /[{(]/; var Ke = /[})]/; function Ye(e) { var t = []; var r = 0; while (r < e.length) { if ($e.exec(e.charAt(r))) { var n = r; while (We.exec(e.charAt(r + 1))) { r++ } t.push(e.substr(n, r - n + 1)) } else if (Ge.indexOf(e.charAt(r)) !== -1) { var i = e.charAt(r); var n = r; r++; while (r < e.length && e.charAt(r) !== i) { if (e.charAt(r) === "\\") { r++ } r++ } t.push(e.substr(n, r - n + 1)) } else { var a = e.charAt(r); t.push(a) } r++ } return t } function Qe(e, t, r) { return $e.exec(e.charAt(0)) && e !== "true" && e !== "false" && e !== "this" && e !== r && t !== "." } function et(e, t, r) { if (t[0] === "[") { t.shift(); var n = 1; var i = " return (function(" + r + "){ return ("; var a = null; while (t.length > 0) { var o = t[0]; if (o === "]") { n--; if (n === 0) { if (a === null) { i = i + "true" } t.shift(); i += ")})"; try { var s = Tr(e, function () { return Function(i)() }, function () { return true }); s.source = i; return s } catch (e) { fe(re().body, "htmx:syntax:error", { error: e, source: i }); return null } } } else if (o === "[") { n++ } if (Qe(o, a, r)) { i += "((" + r + "." + o + ") ? (" + r + "." + o + ") : (window." + o + "))" } else { i = i + o } a = t.shift() } } } function y(e, t) { var r = ""; while (e.length > 0 && !t.test(e[0])) { r += e.shift() } return r } function tt(e) { var t; if (e.length > 0 && Ze.test(e[0])) { e.shift(); t = y(e, Ke).trim(); e.shift() } else { t = y(e, x) } return t } var rt = "input, textarea, select"; function nt(e, t, r) { var n = []; var i = Ye(t); do { y(i, Je); var a = i.length; var o = y(i, /[,\[\s]/); if (o !== "") { if (o === "every") { var s = { trigger: "every" }; y(i, Je); s.pollInterval = d(y(i, /[,\[\s]/)); y(i, Je); var l = et(e, i, "event"); if (l) { s.eventFilter = l } n.push(s) } else if (o.indexOf("sse:") === 0) { n.push({ trigger: "sse", sseEvent: o.substr(4) }) } else { var u = { trigger: o }; var l = et(e, i, "event"); if (l) { u.eventFilter = l } while (i.length > 0 && i[0] !== ",") { y(i, Je); var f = i.shift(); if (f === "changed") { u.changed = true } else if (f === "once") { u.once = true } else if (f === "consume") { u.consume = true } else if (f === "delay" && i[0] === ":") { i.shift(); u.delay = d(y(i, x)) } else if (f === "from" && i[0] === ":") { i.shift(); if (Ze.test(i[0])) { var c = tt(i) } else { var c = y(i, x); if (c === "closest" || c === "find" || c === "next" || c === "previous") { i.shift(); var h = tt(i); if (h.length > 0) { c += " " + h } } } u.from = c } else if (f === "target" && i[0] === ":") { i.shift(); u.target = tt(i) } else if (f === "throttle" && i[0] === ":") { i.shift(); u.throttle = d(y(i, x)) } else if (f === "queue" && i[0] === ":") { i.shift(); u.queue = y(i, x) } else if (f === "root" && i[0] === ":") { i.shift(); u[f] = tt(i) } else if (f === "threshold" && i[0] === ":") { i.shift(); u[f] = y(i, x) } else { fe(e, "htmx:syntax:error", { token: i.shift() }) } } n.push(u) } } if (i.length === a) { fe(e, "htmx:syntax:error", { token: i.shift() }) } y(i, Je) } while (i[0] === "," && i.shift()); if (r) { r[t] = n } return n } function it(e) { var t = te(e, "hx-trigger"); var r = []; if (t) { var n = Q.config.triggerSpecsCache; r = n && n[t] || nt(e, t, n) } if (r.length > 0) { return r } else if (h(e, "form")) { return [{ trigger: "submit" }] } else if (h(e, 'input[type="button"], input[type="submit"]')) { return [{ trigger: "click" }] } else if (h(e, rt)) { return [{ trigger: "change" }] } else { return [{ trigger: "click" }] } } function at(e) { ae(e).cancelled = true } function ot(e, t, r) { var n = ae(e); n.timeout = setTimeout(function () { if (se(e) && n.cancelled !== true) { if (!ct(r, e, Wt("hx:poll:trigger", { triggerSpec: r, target: e }))) { t(e) } ot(e, t, r) } }, r.pollInterval) } function st(e) { return location.hostname === e.hostname && ee(e, "href") && ee(e, "href").indexOf("#") !== 0 } function lt(t, r, e) { if (t.tagName === "A" && st(t) && (t.target === "" || t.target === "_self") || t.tagName === "FORM") { r.boosted = true; var n, i; if (t.tagName === "A") { n = "get"; i = ee(t, "href") } else { var a = ee(t, "method"); n = a ? a.toLowerCase() : "get"; if (n === "get") { } i = ee(t, "action") } e.forEach(function (e) { ht(t, function (e, t) { if (v(e, Q.config.disableSelector)) { p(e); return } he(n, i, e, t) }, r, e, true) }) } } function ut(e, t) { if (e.type === "submit" || e.type === "click") { if (t.tagName === "FORM") { return true } if (h(t, 'input[type="submit"], button') && v(t, "form") !== null) { return true } if (t.tagName === "A" && t.href && (t.getAttribute("href") === "#" || t.getAttribute("href").indexOf("#") !== 0)) { return true } } return false } function ft(e, t) { return ae(e).boosted && e.tagName === "A" && t.type === "click" && (t.ctrlKey || t.metaKey) } function ct(e, t, r) { var n = e.eventFilter; if (n) { try { return n.call(t, r) !== true } catch (e) { fe(re().body, "htmx:eventFilter:error", { error: e, source: n.source }); return true } } return false } function ht(a, o, e, s, l) { var u = ae(a); var t; if (s.from) { t = Z(a, s.from) } else { t = [a] } if (s.changed) { t.forEach(function (e) { var t = ae(e); t.lastValue = e.value }) } oe(t, function (n) { var i = function (e) { if (!se(a)) { n.removeEventListener(s.trigger, i); return } if (ft(a, e)) { return } if (l || ut(e, a)) { e.preventDefault() } if (ct(s, a, e)) { return } var t = ae(e); t.triggerSpec = s; if (t.handledFor == null) { t.handledFor = [] } if (t.handledFor.indexOf(a) < 0) { t.handledFor.push(a); if (s.consume) { e.stopPropagation() } if (s.target && e.target) { if (!h(e.target, s.target)) { return } } if (s.once) { if (u.triggeredOnce) { return } else { u.triggeredOnce = true } } if (s.changed) { var r = ae(n); if (r.lastValue === n.value) { return } r.lastValue = n.value } if (u.delayed) { clearTimeout(u.delayed) } if (u.throttle) { return } if (s.throttle > 0) { if (!u.throttle) { o(a, e); u.throttle = setTimeout(function () { u.throttle = null }, s.throttle) } } else if (s.delay > 0) { u.delayed = setTimeout(function () { o(a, e) }, s.delay) } else { ce(a, "htmx:trigger"); o(a, e) } } }; if (e.listenerInfos == null) { e.listenerInfos = [] } e.listenerInfos.push({ trigger: s.trigger, listener: i, on: n }); n.addEventListener(s.trigger, i) }) } var vt = false; var dt = null; function gt() { if (!dt) { dt = function () { vt = true }; window.addEventListener("scroll", dt); setInterval(function () { if (vt) { vt = false; oe(re().querySelectorAll("[hx-trigger='revealed'],[data-hx-trigger='revealed']"), function (e) { mt(e) }) } }, 200) } } function mt(t) { if (!o(t, "data-hx-revealed") && X(t)) { t.setAttribute("data-hx-revealed", "true"); var e = ae(t); if (e.initHash) { ce(t, "revealed") } else { t.addEventListener("htmx:afterProcessNode", function (e) { ce(t, "revealed") }, { once: true }) } } } function pt(e, t, r) { var n = D(r); for (var i = 0; i < n.length; i++) { var a = n[i].split(/:(.+)/); if (a[0] === "connect") { xt(e, a[1], 0) } if (a[0] === "send") { bt(e) } } } function xt(s, r, n) { if (!se(s)) { return } if (r.indexOf("/") == 0) { var e = location.hostname + (location.port ? ":" + location.port : ""); if (location.protocol == "https:") { r = "wss://" + e + r } else if (location.protocol == "http:") { r = "ws://" + e + r } } var t = Q.createWebSocket(r); t.onerror = function (e) { fe(s, "htmx:wsError", { error: e, socket: t }); yt(s) }; t.onclose = function (e) { if ([1006, 1012, 1013].indexOf(e.code) >= 0) { var t = wt(n); setTimeout(function () { xt(s, r, n + 1) }, t) } }; t.onopen = function (e) { n = 0 }; ae(s).webSocket = t; t.addEventListener("message", function (e) { if (yt(s)) { return } var t = e.data; R(s, function (e) { t = e.transformResponse(t, null, s) }); var r = T(s); var n = l(t); var i = M(n.children); for (var a = 0; a < i.length; a++) { var o = i[a]; Ee(te(o, "hx-swap-oob") || "true", o, r) } nr(r.tasks) }) } function yt(e) { if (!se(e)) { ae(e).webSocket.close(); return true } } function bt(u) { var f = c(u, function (e) { return ae(e).webSocket != null }); if (f) { u.addEventListener(it(u)[0].trigger, function (e) { var t = ae(f).webSocket; var r = xr(u, f); var n = dr(u, "post"); var i = n.errors; var a = n.values; var o = Hr(u); var s = le(a, o); var l = yr(s, u); l["HEADERS"] = r; if (i && i.length > 0) { ce(u, "htmx:validation:halted", i); return } t.send(JSON.stringify(l)); if (ut(e, u)) { e.preventDefault() } }) } else { fe(u, "htmx:noWebSocketSourceError") } } function wt(e) { var t = Q.config.wsReconnectDelay; if (typeof t === "function") { return t(e) } if (t === "full-jitter") { var r = Math.min(e, 6); var n = 1e3 * Math.pow(2, r); return n * Math.random() } b('htmx.config.wsReconnectDelay must either be a function or the string "full-jitter"') } function St(e, t, r) { var n = D(r); for (var i = 0; i < n.length; i++) { var a = n[i].split(/:(.+)/); if (a[0] === "connect") { Et(e, a[1]) } if (a[0] === "swap") { Ct(e, a[1]) } } } function Et(t, e) { var r = Q.createEventSource(e); r.onerror = function (e) { fe(t, "htmx:sseError", { error: e, source: r }); Tt(t) }; ae(t).sseEventSource = r } function Ct(a, o) { var s = c(a, Ot); if (s) { var l = ae(s).sseEventSource; var u = function (e) { if (Tt(s)) { return } if (!se(a)) { l.removeEventListener(o, u); return } var t = e.data; R(a, function (e) { t = e.transformResponse(t, null, a) }); var r = wr(a); var n = ye(a); var i = T(a); je(r.swapStyle, n, a, t, i); nr(i.tasks); ce(a, "htmx:sseMessage", e) }; ae(a).sseListener = u; l.addEventListener(o, u) } else { fe(a, "htmx:noSSESourceError") } } function Rt(e, t, r) { var n = c(e, Ot); if (n) { var i = ae(n).sseEventSource; var a = function () { if (!Tt(n)) { if (se(e)) { t(e) } else { i.removeEventListener(r, a) } } }; ae(e).sseListener = a; i.addEventListener(r, a) } else { fe(e, "htmx:noSSESourceError") } } function Tt(e) { if (!se(e)) { ae(e).sseEventSource.close(); return true } } function Ot(e) { return ae(e).sseEventSource != null } function qt(e, t, r, n) { var i = function () { if (!r.loaded) { r.loaded = true; t(e) } }; if (n > 0) { setTimeout(i, n) } else { i() } } function Ht(t, i, e) { var a = false; oe(w, function (r) { if (o(t, "hx-" + r)) { var n = te(t, "hx-" + r); a = true; i.path = n; i.verb = r; e.forEach(function (e) { Lt(t, e, i, function (e, t) { if (v(e, Q.config.disableSelector)) { p(e); return } he(r, n, e, t) }) }) } }); return a } function Lt(n, e, t, r) { if (e.sseEvent) { Rt(n, r, e.sseEvent) } else if (e.trigger === "revealed") { gt(); ht(n, r, t, e); mt(n) } else if (e.trigger === "intersect") { var i = {}; if (e.root) { i.root = ue(n, e.root) } if (e.threshold) { i.threshold = parseFloat(e.threshold) } var a = new IntersectionObserver(function (e) { for (var t = 0; t < e.length; t++) { var r = e[t]; if (r.isIntersecting) { ce(n, "intersect"); break } } }, i); a.observe(n); ht(n, r, t, e) } else if (e.trigger === "load") { if (!ct(e, n, Wt("load", { elt: n }))) { qt(n, r, t, e.delay) } } else if (e.pollInterval > 0) { t.polling = true; ot(n, r, e) } else { ht(n, r, t, e) } } function At(e) { if (Q.config.allowScriptTags && (e.type === "text/javascript" || e.type === "module" || e.type === "")) { var t = re().createElement("script"); oe(e.attributes, function (e) { t.setAttribute(e.name, e.value) }); t.textContent = e.textContent; t.async = false; if (Q.config.inlineScriptNonce) { t.nonce = Q.config.inlineScriptNonce } var r = e.parentElement; try { r.insertBefore(t, e) } catch (e) { b(e) } finally { if (e.parentElement) { e.parentElement.removeChild(e) } } } } function Nt(e) { if (h(e, "script")) { At(e) } oe(f(e, "script"), function (e) { At(e) }) } function It(e) { var t = e.attributes; for (var r = 0; r < t.length; r++) { var n = t[r].name; if (s(n, "hx-on:") || s(n, "data-hx-on:") || s(n, "hx-on-") || s(n, "data-hx-on-")) { return true } } return false } function kt(e) { var t = null; var r = []; if (It(e)) { r.push(e) } if (document.evaluate) { var n = document.evaluate('.//*[@*[ starts-with(name(), "hx-on:") or starts-with(name(), "data-hx-on:") or' + ' starts-with(name(), "hx-on-") or starts-with(name(), "data-hx-on-") ]]', e); while (t = n.iterateNext()) r.push(t) } else { var i = e.getElementsByTagName("*"); for (var a = 0; a < i.length; a++) { if (It(i[a])) { r.push(i[a]) } } } return r } function Pt(e) { if (e.querySelectorAll) { var t = ", [hx-boost] a, [data-hx-boost] a, a[hx-boost], a[data-hx-boost]"; var r = e.querySelectorAll(i + t + ", form, [type='submit'], [hx-sse], [data-hx-sse], [hx-ws]," + " [data-hx-ws], [hx-ext], [data-hx-ext], [hx-trigger], [data-hx-trigger], [hx-on], [data-hx-on]"); return r } else { return [] } } function Mt(e) { var t = v(e.target, "button, input[type='submit']"); var r = Dt(e); if (r) { r.lastButtonClicked = t } } function Xt(e) { var t = Dt(e); if (t) { t.lastButtonClicked = null } } function Dt(e) { var t = v(e.target, "button, input[type='submit']"); if (!t) { return } var r = g("#" + ee(t, "form")) || v(t, "form"); if (!r) { return } return ae(r) } function Ut(e) { e.addEventListener("click", Mt); e.addEventListener("focusin", Mt); e.addEventListener("focusout", Xt) } function Bt(e) { var t = Ye(e); var r = 0; for (var n = 0; n < t.length; n++) { const i = t[n]; if (i === "{") { r++ } else if (i === "}") { r-- } } return r } function Ft(t, e, r) { var n = ae(t); if (!Array.isArray(n.onHandlers)) { n.onHandlers = [] } var i; var a = function (e) { return Tr(t, function () { if (!i) { i = new Function("event", r) } i.call(t, e) }) }; t.addEventListener(e, a); n.onHandlers.push({ event: e, listener: a }) } function Vt(e) { var t = te(e, "hx-on"); if (t) { var r = {}; var n = t.split("\n"); var i = null; var a = 0; while (n.length > 0) { var o = n.shift(); var s = o.match(/^\s*([a-zA-Z:\-\.]+:)(.*)/); if (a === 0 && s) { o.split(":"); i = s[1].slice(0, -1); r[i] = s[2] } else { r[i] += o } a += Bt(o) } for (var l in r) { Ft(e, l, r[l]) } } } function jt(e) { Ae(e); for (var t = 0; t < e.attributes.length; t++) { var r = e.attributes[t].name; var n = e.attributes[t].value; if (s(r, "hx-on") || s(r, "data-hx-on")) { var i = r.indexOf("-on") + 3; var a = r.slice(i, i + 1); if (a === "-" || a === ":") { var o = r.slice(i + 1); if (s(o, ":")) { o = "htmx" + o } else if (s(o, "-")) { o = "htmx:" + o.slice(1) } else if (s(o, "htmx-")) { o = "htmx:" + o.slice(5) } Ft(e, o, n) } } } } function _t(t) { if (v(t, Q.config.disableSelector)) { p(t); return } var r = ae(t); if (r.initHash !== Le(t)) { Ne(t); r.initHash = Le(t); Vt(t); ce(t, "htmx:beforeProcessNode"); if (t.value) { r.lastValue = t.value } var e = it(t); var n = Ht(t, r, e); if (!n) { if (ne(t, "hx-boost") === "true") { lt(t, r, e) } else if (o(t, "hx-trigger")) { e.forEach(function (e) { Lt(t, e, r, function () { }) }) } } if (t.tagName === "FORM" || ee(t, "type") === "submit" && o(t, "form")) { Ut(t) } var i = te(t, "hx-sse"); if (i) { St(t, r, i) } var a = te(t, "hx-ws"); if (a) { pt(t, r, a) } ce(t, "htmx:afterProcessNode") } } function zt(e) { e = g(e); if (v(e, Q.config.disableSelector)) { p(e); return } _t(e); oe(Pt(e), function (e) { _t(e) }); oe(kt(e), jt) } function $t(e) { return e.replace(/([a-z0-9])([A-Z])/g, "$1-$2").toLowerCase() } function Wt(e, t) { var r; if (window.CustomEvent && typeof window.CustomEvent === "function") { r = new CustomEvent(e, { bubbles: true, cancelable: true, detail: t }) } else { r = re().createEvent("CustomEvent"); r.initCustomEvent(e, true, true, t) } return r } function fe(e, t, r) { ce(e, t, le({ error: t }, r)) } function Gt(e) { return e === "htmx:afterProcessNode" } function R(e, t) { oe(Fr(e), function (e) { try { t(e) } catch (e) { b(e) } }) } function b(e) { if (console.error) { console.error(e) } else if (console.log) { console.log("ERROR: ", e) } } function ce(e, t, r) { e = g(e); if (r == null) { r = {} } r["elt"] = e; var n = Wt(t, r); if (Q.logger && !Gt(t)) { Q.logger(e, t, r) } if (r.error) { b(r.error); ce(e, "htmx:error", { errorInfo: r }) } var i = e.dispatchEvent(n); var a = $t(t); if (i && a !== t) { var o = Wt(a, n.detail); i = i && e.dispatchEvent(o) } R(e, function (e) { i = i && (e.onEvent(t, n) !== false && !n.defaultPrevented) }); return i } var Jt = location.pathname + location.search; function Zt() { var e = re().querySelector("[hx-history-elt],[data-hx-history-elt]"); return e || re().body } function Kt(e, t, r, n) { if (!U()) { return } if (Q.config.historyCacheSize <= 0) { localStorage.removeItem("htmx-history-cache"); return } e = B(e); var i = E(localStorage.getItem("htmx-history-cache")) || []; for (var a = 0; a < i.length; a++) { if (i[a].url === e) { i.splice(a, 1); break } } var o = { url: e, content: t, title: r, scroll: n }; ce(re().body, "htmx:historyItemCreated", { item: o, cache: i }); i.push(o); while (i.length > Q.config.historyCacheSize) { i.shift() } while (i.length > 0) { try { localStorage.setItem("htmx-history-cache", JSON.stringify(i)); break } catch (e) { fe(re().body, "htmx:historyCacheError", { cause: e, cache: i }); i.shift() } } } function Yt(e) { if (!U()) { return null } e = B(e); var t = E(localStorage.getItem("htmx-history-cache")) || []; for (var r = 0; r < t.length; r++) { if (t[r].url === e) { return t[r] } } return null } function Qt(e) { var t = Q.config.requestClass; var r = e.cloneNode(true); oe(f(r, "." + t), function (e) { n(e, t) }); return r.innerHTML } function er() { var e = Zt(); var t = Jt || location.pathname + location.search; var r; try { r = re().querySelector('[hx-history="false" i],[data-hx-history="false" i]') } catch (e) { r = re().querySelector('[hx-history="false"],[data-hx-history="false"]') } if (!r) { ce(re().body, "htmx:beforeHistorySave", { path: t, historyElt: e }); Kt(t, Qt(e), re().title, window.scrollY) } if (Q.config.historyEnabled) history.replaceState({ htmx: true }, re().title, window.location.href) } function tr(e) { if (Q.config.getCacheBusterParam) { e = e.replace(/org\.htmx\.cache-buster=[^&]*&?/, ""); if (G(e, "&") || G(e, "?")) { e = e.slice(0, -1) } } if (Q.config.historyEnabled) { history.pushState({ htmx: true }, "", e) } Jt = e } function rr(e) { if (Q.config.historyEnabled) history.replaceState({ htmx: true }, "", e); Jt = e } function nr(e) { oe(e, function (e) { e.call() }) } function ir(a) { var e = new XMLHttpRequest; var o = { path: a, xhr: e }; ce(re().body, "htmx:historyCacheMiss", o); e.open("GET", a, true); e.setRequestHeader("HX-Request", "true"); e.setRequestHeader("HX-History-Restore-Request", "true"); e.setRequestHeader("HX-Current-URL", re().location.href); e.onload = function () { if (this.status >= 200 && this.status < 400) { ce(re().body, "htmx:historyCacheMissLoad", o); var e = l(this.response); e = e.querySelector("[hx-history-elt],[data-hx-history-elt]") || e; var t = Zt(); var r = T(t); var n = Ve(this.response); if (n) { var i = C("title"); if (i) { i.innerHTML = n } else { window.document.title = n } } Ue(t, e, r); nr(r.tasks); Jt = a; ce(re().body, "htmx:historyRestore", { path: a, cacheMiss: true, serverResponse: this.response }) } else { fe(re().body, "htmx:historyCacheMissLoadError", o) } }; e.send() } function ar(e) { er(); e = e || location.pathname + location.search; var t = Yt(e); if (t) { var r = l(t.content); var n = Zt(); var i = T(n); Ue(n, r, i); nr(i.tasks); document.title = t.title; setTimeout(function () { window.scrollTo(0, t.scroll) }, 0); Jt = e; ce(re().body, "htmx:historyRestore", { path: e, item: t }) } else { if (Q.config.refreshOnHistoryMiss) { window.location.reload(true) } else { ir(e) } } } function or(e) { var t = pe(e, "hx-indicator"); if (t == null) { t = [e] } oe(t, function (e) { var t = ae(e); t.requestCount = (t.requestCount || 0) + 1; e.classList["add"].call(e.classList, Q.config.requestClass) }); return t } function sr(e) { var t = pe(e, "hx-disabled-elt"); if (t == null) { t = [] } oe(t, function (e) { var t = ae(e); t.requestCount = (t.requestCount || 0) + 1; e.setAttribute("disabled", "") }); return t } function lr(e, t) { oe(e, function (e) { var t = ae(e); t.requestCount = (t.requestCount || 0) - 1; if (t.requestCount === 0) { e.classList["remove"].call(e.classList, Q.config.requestClass) } }); oe(t, function (e) { var t = ae(e); t.requestCount = (t.requestCount || 0) - 1; if (t.requestCount === 0) { e.removeAttribute("disabled") } }) } function ur(e, t) { for (var r = 0; r < e.length; r++) { var n = e[r]; if (n.isSameNode(t)) { return true } } return false } function fr(e) { if (e.name === "" || e.name == null || e.disabled || v(e, "fieldset[disabled]")) { return false } if (e.type === "button" || e.type === "submit" || e.tagName === "image" || e.tagName === "reset" || e.tagName === "file") { return false } if (e.type === "checkbox" || e.type === "radio") { return e.checked } return true } function cr(e, t, r) { if (e != null && t != null) { var n = r[e]; if (n === undefined) { r[e] = t } else if (Array.isArray(n)) { if (Array.isArray(t)) { r[e] = n.concat(t) } else { n.push(t) } } else { if (Array.isArray(t)) { r[e] = [n].concat(t) } else { r[e] = [n, t] } } } } function hr(t, r, n, e, i) { if (e == null || ur(t, e)) { return } else { t.push(e) } if (fr(e)) { var a = ee(e, "name"); var o = e.value; if (e.multiple && e.tagName === "SELECT") { o = M(e.querySelectorAll("option:checked")).map(function (e) { return e.value }) } if (e.files) { o = M(e.files) } cr(a, o, r); if (i) { vr(e, n) } } if (h(e, "form")) { var s = e.elements; oe(s, function (e) { hr(t, r, n, e, i) }) } } function vr(e, t) { if (e.willValidate) { ce(e, "htmx:validation:validate"); if (!e.checkValidity()) { t.push({ elt: e, message: e.validationMessage, validity: e.validity }); ce(e, "htmx:validation:failed", { message: e.validationMessage, validity: e.validity }) } } } function dr(e, t) { var r = []; var n = {}; var i = {}; var a = []; var o = ae(e); if (o.lastButtonClicked && !se(o.lastButtonClicked)) { o.lastButtonClicked = null } var s = h(e, "form") && e.noValidate !== true || te(e, "hx-validate") === "true"; if (o.lastButtonClicked) { s = s && o.lastButtonClicked.formNoValidate !== true } if (t !== "get") { hr(r, i, a, v(e, "form"), s) } hr(r, n, a, e, s); if (o.lastButtonClicked || e.tagName === "BUTTON" || e.tagName === "INPUT" && ee(e, "type") === "submit") { var l = o.lastButtonClicked || e; var u = ee(l, "name"); cr(u, l.value, i) } var f = pe(e, "hx-include"); oe(f, function (e) { hr(r, n, a, e, s); if (!h(e, "form")) { oe(e.querySelectorAll(rt), function (e) { hr(r, n, a, e, s) }) } }); n = le(n, i); return { errors: a, values: n } } function gr(e, t, r) { if (e !== "") { e += "&" } if (String(r) === "[object Object]") { r = JSON.stringify(r) } var n = encodeURIComponent(r); e += encodeURIComponent(t) + "=" + n; return e } function mr(e) { var t = ""; for (var r in e) { if (e.hasOwnProperty(r)) { var n = e[r]; if (Array.isArray(n)) { oe(n, function (e) { t = gr(t, r, e) }) } else { t = gr(t, r, n) } } } return t } function pr(e) { var t = new FormData; for (var r in e) { if (e.hasOwnProperty(r)) { var n = e[r]; if (Array.isArray(n)) { oe(n, function (e) { t.append(r, e) }) } else { t.append(r, n) } } } return t } function xr(e, t, r) { var n = { "HX-Request": "true", "HX-Trigger": ee(e, "id"), "HX-Trigger-Name": ee(e, "name"), "HX-Target": te(t, "id"), "HX-Current-URL": re().location.href }; Rr(e, "hx-headers", false, n); if (r !== undefined) { n["HX-Prompt"] = r } if (ae(e).boosted) { n["HX-Boosted"] = "true" } return n } function yr(t, e) { var r = ne(e, "hx-params"); if (r) { if (r === "none") { return {} } else if (r === "*") { return t } else if (r.indexOf("not ") === 0) { oe(r.substr(4).split(","), function (e) { e = e.trim(); delete t[e] }); return t } else { var n = {}; oe(r.split(","), function (e) { e = e.trim(); n[e] = t[e] }); return n } } else { return t } } function br(e) { return ee(e, "href") && ee(e, "href").indexOf("#") >= 0 } function wr(e, t) { var r = t ? t : ne(e, "hx-swap"); var n = { swapStyle: ae(e).boosted ? "innerHTML" : Q.config.defaultSwapStyle, swapDelay: Q.config.defaultSwapDelay, settleDelay: Q.config.defaultSettleDelay }; if (Q.config.scrollIntoViewOnBoost && ae(e).boosted && !br(e)) { n["show"] = "top" } if (r) { var i = D(r); if (i.length > 0) { for (var a = 0; a < i.length; a++) { var o = i[a]; if (o.indexOf("swap:") === 0) { n["swapDelay"] = d(o.substr(5)) } else if (o.indexOf("settle:") === 0) { n["settleDelay"] = d(o.substr(7)) } else if (o.indexOf("transition:") === 0) { n["transition"] = o.substr(11) === "true" } else if (o.indexOf("ignoreTitle:") === 0) { n["ignoreTitle"] = o.substr(12) === "true" } else if (o.indexOf("scroll:") === 0) { var s = o.substr(7); var l = s.split(":"); var u = l.pop(); var f = l.length > 0 ? l.join(":") : null; n["scroll"] = u; n["scrollTarget"] = f } else if (o.indexOf("show:") === 0) { var c = o.substr(5); var l = c.split(":"); var h = l.pop(); var f = l.length > 0 ? l.join(":") : null; n["show"] = h; n["showTarget"] = f } else if (o.indexOf("focus-scroll:") === 0) { var v = o.substr("focus-scroll:".length); n["focusScroll"] = v == "true" } else if (a == 0) { n["swapStyle"] = o } else { b("Unknown modifier in hx-swap: " + o) } } } } return n } function Sr(e) { return ne(e, "hx-encoding") === "multipart/form-data" || h(e, "form") && ee(e, "enctype") === "multipart/form-data" } function Er(t, r, n) { var i = null; R(r, function (e) { if (i == null) { i = e.encodeParameters(t, n, r) } }); if (i != null) { return i } else { if (Sr(r)) { return pr(n) } else { return mr(n) } } } function T(e) { return { tasks: [], elts: [e] } } function Cr(e, t) { var r = e[0]; var n = e[e.length - 1]; if (t.scroll) { var i = null; if (t.scrollTarget) { i = ue(r, t.scrollTarget) } if (t.scroll === "top" && (r || i)) { i = i || r; i.scrollTop = 0 } if (t.scroll === "bottom" && (n || i)) { i = i || n; i.scrollTop = i.scrollHeight } } if (t.show) { var i = null; if (t.showTarget) { var a = t.showTarget; if (t.showTarget === "window") { a = "body" } i = ue(r, a) } if (t.show === "top" && (r || i)) { i = i || r; i.scrollIntoView({ block: "start", behavior: Q.config.scrollBehavior }) } if (t.show === "bottom" && (n || i)) { i = i || n; i.scrollIntoView({ block: "end", behavior: Q.config.scrollBehavior }) } } } function Rr(e, t, r, n) { if (n == null) { n = {} } if (e == null) { return n } var i = te(e, t); if (i) { var a = i.trim(); var o = r; if (a === "unset") { return null } if (a.indexOf("javascript:") === 0) { a = a.substr(11); o = true } else if (a.indexOf("js:") === 0) { a = a.substr(3); o = true } if (a.indexOf("{") !== 0) { a = "{" + a + "}" } var s; if (o) { s = Tr(e, function () { return Function("return (" + a + ")")() }, {}) } else { s = E(a) } for (var l in s) { if (s.hasOwnProperty(l)) { if (n[l] == null) { n[l] = s[l] } } } } return Rr(u(e), t, r, n) } function Tr(e, t, r) { if (Q.config.allowEval) { return t() } else { fe(e, "htmx:evalDisallowedError"); return r } } function Or(e, t) { return Rr(e, "hx-vars", true, t) } function qr(e, t) { return Rr(e, "hx-vals", false, t) } function Hr(e) { return le(Or(e), qr(e)) } function Lr(t, r, n) { if (n !== null) { try { t.setRequestHeader(r, n) } catch (e) { t.setRequestHeader(r, encodeURIComponent(n)); t.setRequestHeader(r + "-URI-AutoEncoded", "true") } } } function Ar(t) { if (t.responseURL && typeof URL !== "undefined") { try { var e = new URL(t.responseURL); return e.pathname + e.search } catch (e) { fe(re().body, "htmx:badResponseUrl", { url: t.responseURL }) } } } function O(e, t) { return t.test(e.getAllResponseHeaders()) } function Nr(e, t, r) { e = e.toLowerCase(); if (r) { if (r instanceof Element || I(r, "String")) { return he(e, t, null, null, { targetOverride: g(r), returnPromise: true }) } else { return he(e, t, g(r.source), r.event, { handler: r.handler, headers: r.headers, values: r.values, targetOverride: g(r.target), swapOverride: r.swap, select: r.select, returnPromise: true }) } } else { return he(e, t, null, null, { returnPromise: true }) } } function Ir(e) { var t = []; while (e) { t.push(e); e = e.parentElement } return t } function kr(e, t, r) { var n; var i; if (typeof URL === "function") { i = new URL(t, document.location.href); var a = document.location.origin; n = a === i.origin } else { i = t; n = s(t, document.location.origin) } if (Q.config.selfRequestsOnly) { if (!n) { return false } } return ce(e, "htmx:validateUrl", le({ url: i, sameHost: n }, r)) } function he(t, r, n, i, a, e) { var o = null; var s = null; a = a != null ? a : {}; if (a.returnPromise && typeof Promise !== "undefined") { var l = new Promise(function (e, t) { o = e; s = t }) } if (n == null) { n = re().body } var M = a.handler || Mr; var X = a.select || null; if (!se(n)) { ie(o); return l } var u = a.targetOverride || ye(n); if (u == null || u == me) { fe(n, "htmx:targetError", { target: te(n, "hx-target") }); ie(s); return l } var f = ae(n); var c = f.lastButtonClicked; if (c) { var h = ee(c, "formaction"); if (h != null) { r = h } var v = ee(c, "formmethod"); if (v != null) { if (v.toLowerCase() !== "dialog") { t = v } } } var d = ne(n, "hx-confirm"); if (e === undefined) { var D = function (e) { return he(t, r, n, i, a, !!e) }; var U = { target: u, elt: n, path: r, verb: t, triggeringEvent: i, etc: a, issueRequest: D, question: d }; if (ce(n, "htmx:confirm", U) === false) { ie(o); return l } } var g = n; var m = ne(n, "hx-sync"); var p = null; var x = false; if (m) { var B = m.split(":"); var F = B[0].trim(); if (F === "this") { g = xe(n, "hx-sync") } else { g = ue(n, F) } m = (B[1] || "drop").trim(); f = ae(g); if (m === "drop" && f.xhr && f.abortable !== true) { ie(o); return l } else if (m === "abort") { if (f.xhr) { ie(o); return l } else { x = true } } else if (m === "replace") { ce(g, "htmx:abort") } else if (m.indexOf("queue") === 0) { var V = m.split(" "); p = (V[1] || "last").trim() } } if (f.xhr) { if (f.abortable) { ce(g, "htmx:abort") } else { if (p == null) { if (i) { var y = ae(i); if (y && y.triggerSpec && y.triggerSpec.queue) { p = y.triggerSpec.queue } } if (p == null) { p = "last" } } if (f.queuedRequests == null) { f.queuedRequests = [] } if (p === "first" && f.queuedRequests.length === 0) { f.queuedRequests.push(function () { he(t, r, n, i, a) }) } else if (p === "all") { f.queuedRequests.push(function () { he(t, r, n, i, a) }) } else if (p === "last") { f.queuedRequests = []; f.queuedRequests.push(function () { he(t, r, n, i, a) }) } ie(o); return l } } var b = new XMLHttpRequest; f.xhr = b; f.abortable = x; var w = function () { f.xhr = null; f.abortable = false; if (f.queuedRequests != null && f.queuedRequests.length > 0) { var e = f.queuedRequests.shift(); e() } }; var j = ne(n, "hx-prompt"); if (j) { var S = prompt(j); if (S === null || !ce(n, "htmx:prompt", { prompt: S, target: u })) { ie(o); w(); return l } } if (d && !e) { if (!confirm(d)) { ie(o); w(); return l } } var E = xr(n, u, S); if (t !== "get" && !Sr(n)) { E["Content-Type"] = "application/x-www-form-urlencoded" } if (a.headers) { E = le(E, a.headers) } var _ = dr(n, t); var C = _.errors; var R = _.values; if (a.values) { R = le(R, a.values) } var z = Hr(n); var $ = le(R, z); var T = yr($, n); if (Q.config.getCacheBusterParam && t === "get") { T["org.htmx.cache-buster"] = ee(u, "id") || "true" } if (r == null || r === "") { r = re().location.href } var O = Rr(n, "hx-request"); var W = ae(n).boosted; var q = Q.config.methodsThatUseUrlParams.indexOf(t) >= 0; var H = { boosted: W, useUrlParams: q, parameters: T, unfilteredParameters: $, headers: E, target: u, verb: t, errors: C, withCredentials: a.credentials || O.credentials || Q.config.withCredentials, timeout: a.timeout || O.timeout || Q.config.timeout, path: r, triggeringEvent: i }; if (!ce(n, "htmx:configRequest", H)) { ie(o); w(); return l } r = H.path; t = H.verb; E = H.headers; T = H.parameters; C = H.errors; q = H.useUrlParams; if (C && C.length > 0) { ce(n, "htmx:validation:halted", H); ie(o); w(); return l } var G = r.split("#"); var J = G[0]; var L = G[1]; var A = r; if (q) { A = J; var Z = Object.keys(T).length !== 0; if (Z) { if (A.indexOf("?") < 0) { A += "?" } else { A += "&" } A += mr(T); if (L) { A += "#" + L } } } if (!kr(n, A, H)) { fe(n, "htmx:invalidPath", H); ie(s); return l } b.open(t.toUpperCase(), A, true); b.overrideMimeType("text/html"); b.withCredentials = H.withCredentials; b.timeout = H.timeout; if (O.noHeaders) { } else { for (var N in E) { if (E.hasOwnProperty(N)) { var K = E[N]; Lr(b, N, K) } } } var I = { xhr: b, target: u, requestConfig: H, etc: a, boosted: W, select: X, pathInfo: { requestPath: r, finalRequestPath: A, anchor: L } }; b.onload = function () { try { var e = Ir(n); I.pathInfo.responsePath = Ar(b); M(n, I); lr(k, P); ce(n, "htmx:afterRequest", I); ce(n, "htmx:afterOnLoad", I); if (!se(n)) { var t = null; while (e.length > 0 && t == null) { var r = e.shift(); if (se(r)) { t = r } } if (t) { ce(t, "htmx:afterRequest", I); ce(t, "htmx:afterOnLoad", I) } } ie(o); w() } catch (e) { fe(n, "htmx:onLoadError", le({ error: e }, I)); throw e } }; b.onerror = function () { lr(k, P); fe(n, "htmx:afterRequest", I); fe(n, "htmx:sendError", I); ie(s); w() }; b.onabort = function () { lr(k, P); fe(n, "htmx:afterRequest", I); fe(n, "htmx:sendAbort", I); ie(s); w() }; b.ontimeout = function () { lr(k, P); fe(n, "htmx:afterRequest", I); fe(n, "htmx:timeout", I); ie(s); w() }; if (!ce(n, "htmx:beforeRequest", I)) { ie(o); w(); return l } var k = or(n); var P = sr(n); oe(["loadstart", "loadend", "progress", "abort"], function (t) { oe([b, b.upload], function (e) { e.addEventListener(t, function (e) { ce(n, "htmx:xhr:" + t, { lengthComputable: e.lengthComputable, loaded: e.loaded, total: e.total }) }) }) }); ce(n, "htmx:beforeSend", I); var Y = q ? null : Er(b, n, T); b.send(Y); return l } function Pr(e, t) { var r = t.xhr; var n = null; var i = null; if (O(r, /HX-Push:/i)) { n = r.getResponseHeader("HX-Push"); i = "push" } else if (O(r, /HX-Push-Url:/i)) { n = r.getResponseHeader("HX-Push-Url"); i = "push" } else if (O(r, /HX-Replace-Url:/i)) { n = r.getResponseHeader("HX-Replace-Url"); i = "replace" } if (n) { if (n === "false") { return {} } else { return { type: i, path: n } } } var a = t.pathInfo.finalRequestPath; var o = t.pathInfo.responsePath; var s = ne(e, "hx-push-url"); var l = ne(e, "hx-replace-url"); var u = ae(e).boosted; var f = null; var c = null; if (s) { f = "push"; c = s } else if (l) { f = "replace"; c = l } else if (u) { f = "push"; c = o || a } if (c) { if (c === "false") { return {} } if (c === "true") { c = o || a } if (t.pathInfo.anchor && c.indexOf("#") === -1) { c = c + "#" + t.pathInfo.anchor } return { type: f, path: c } } else { return {} } } function Mr(l, u) { var f = u.xhr; var c = u.target; var e = u.etc; var t = u.requestConfig; var h = u.select; if (!ce(l, "htmx:beforeOnLoad", u)) return; if (O(f, /HX-Trigger:/i)) { _e(f, "HX-Trigger", l) } if (O(f, /HX-Location:/i)) { er(); var r = f.getResponseHeader("HX-Location"); var v; if (r.indexOf("{") === 0) { v = E(r); r = v["path"]; delete v["path"] } Nr("GET", r, v).then(function () { tr(r) }); return } var n = O(f, /HX-Refresh:/i) && "true" === f.getResponseHeader("HX-Refresh"); if (O(f, /HX-Redirect:/i)) { location.href = f.getResponseHeader("HX-Redirect"); n && location.reload(); return } if (n) { location.reload(); return } if (O(f, /HX-Retarget:/i)) { if (f.getResponseHeader("HX-Retarget") === "this") { u.target = l } else { u.target = ue(l, f.getResponseHeader("HX-Retarget")) } } var d = Pr(l, u); var i = f.status >= 200 && f.status < 400 && f.status !== 204; var g = f.response; var a = f.status >= 400; var m = Q.config.ignoreTitle; var o = le({ shouldSwap: i, serverResponse: g, isError: a, ignoreTitle: m }, u); if (!ce(c, "htmx:beforeSwap", o)) return; c = o.target; g = o.serverResponse; a = o.isError; m = o.ignoreTitle; u.target = c; u.failed = a; u.successful = !a; if (o.shouldSwap) { if (f.status === 286) { at(l) } R(l, function (e) { g = e.transformResponse(g, f, l) }); if (d.type) { er() } var s = e.swapOverride; if (O(f, /HX-Reswap:/i)) { s = f.getResponseHeader("HX-Reswap") } var v = wr(l, s); if (v.hasOwnProperty("ignoreTitle")) { m = v.ignoreTitle } c.classList.add(Q.config.swappingClass); var p = null; var x = null; var y = function () { try { var e = document.activeElement; var t = {}; try { t = { elt: e, start: e ? e.selectionStart : null, end: e ? e.selectionEnd : null } } catch (e) { } var r; if (h) { r = h } if (O(f, /HX-Reselect:/i)) { r = f.getResponseHeader("HX-Reselect") } if (d.type) { ce(re().body, "htmx:beforeHistoryUpdate", le({ history: d }, u)); if (d.type === "push") { tr(d.path); ce(re().body, "htmx:pushedIntoHistory", { path: d.path }) } else { rr(d.path); ce(re().body, "htmx:replacedInHistory", { path: d.path }) } } var n = T(c); je(v.swapStyle, c, l, g, n, r); if (t.elt && !se(t.elt) && ee(t.elt, "id")) { var i = document.getElementById(ee(t.elt, "id")); var a = { preventScroll: v.focusScroll !== undefined ? !v.focusScroll : !Q.config.defaultFocusScroll }; if (i) { if (t.start && i.setSelectionRange) { try { i.setSelectionRange(t.start, t.end) } catch (e) { } } i.focus(a) } } c.classList.remove(Q.config.swappingClass); oe(n.elts, function (e) { if (e.classList) { e.classList.add(Q.config.settlingClass) } ce(e, "htmx:afterSwap", u) }); if (O(f, /HX-Trigger-After-Swap:/i)) { var o = l; if (!se(l)) { o = re().body } _e(f, "HX-Trigger-After-Swap", o) } var s = function () { oe(n.tasks, function (e) { e.call() }); oe(n.elts, function (e) { if (e.classList) { e.classList.remove(Q.config.settlingClass) } ce(e, "htmx:afterSettle", u) }); if (u.pathInfo.anchor) { var e = re().getElementById(u.pathInfo.anchor); if (e) { e.scrollIntoView({ block: "start", behavior: "auto" }) } } if (n.title && !m) { var t = C("title"); if (t) { t.innerHTML = n.title } else { window.document.title = n.title } } Cr(n.elts, v); if (O(f, /HX-Trigger-After-Settle:/i)) { var r = l; if (!se(l)) { r = re().body } _e(f, "HX-Trigger-After-Settle", r) } ie(p) }; if (v.settleDelay > 0) { setTimeout(s, v.settleDelay) } else { s() } } catch (e) { fe(l, "htmx:swapError", u); ie(x); throw e } }; var b = Q.config.globalViewTransitions; if (v.hasOwnProperty("transition")) { b = v.transition } if (b && ce(l, "htmx:beforeTransition", u) && typeof Promise !== "undefined" && document.startViewTransition) { var w = new Promise(function (e, t) { p = e; x = t }); var S = y; y = function () { document.startViewTransition(function () { S(); return w }) } } if (v.swapDelay > 0) { setTimeout(y, v.swapDelay) } else { y() } } if (a) { fe(l, "htmx:responseError", le({ error: "Response Status Error Code " + f.status + " from " + u.pathInfo.requestPath }, u)) } } var Xr = {}; function Dr() { return { init: function (e) { return null }, onEvent: function (e, t) { return true }, transformResponse: function (e, t, r) { return e }, isInlineSwap: function (e) { return false }, handleSwap: function (e, t, r, n) { return false }, encodeParameters: function (e, t, r) { return null } } } function Ur(e, t) { if (t.init) { t.init(r) } Xr[e] = le(Dr(), t) } function Br(e) { delete Xr[e] } function Fr(e, r, n) { if (e == undefined) { return r } if (r == undefined) { r = [] } if (n == undefined) { n = [] } var t = te(e, "hx-ext"); if (t) { oe(t.split(","), function (e) { e = e.replace(/ /g, ""); if (e.slice(0, 7) == "ignore:") { n.push(e.slice(7)); return } if (n.indexOf(e) < 0) { var t = Xr[e]; if (t && r.indexOf(t) < 0) { r.push(t) } } }) } return Fr(u(e), r, n) } var Vr = false; re().addEventListener("DOMContentLoaded", function () { Vr = true }); function jr(e) { if (Vr || re().readyState === "complete") { e() } else { re().addEventListener("DOMContentLoaded", e) } } function _r() { if (Q.config.includeIndicatorStyles !== false) { re().head.insertAdjacentHTML("beforeend", "<style>                      ." + Q.config.indicatorClass + "{opacity:0}                      ." + Q.config.requestClass + " ." + Q.config.indicatorClass + "{opacity:1; transition: opacity 200ms ease-in;}                      ." + Q.config.requestClass + "." + Q.config.indicatorClass + "{opacity:1; transition: opacity 200ms ease-in;}                    </style>") } } function zr() { var e = re().querySelector('meta[name="htmx-config"]'); if (e) { return E(e.content) } else { return null } } function $r() { var e = zr(); if (e) { Q.config = le(Q.config, e) } } jr(function () { $r(); _r(); var e = re().body; zt(e); var t = re().querySelectorAll("[hx-trigger='restored'],[data-hx-trigger='restored']"); e.addEventListener("htmx:abort", function (e) { var t = e.target; var r = ae(t); if (r && r.xhr) { r.xhr.abort() } }); const r = window.onpopstate ? window.onpopstate.bind(window) : null; window.onpopstate = function (e) { if (e.state && e.state.htmx) { ar(); oe(t, function (e) { ce(e, "htmx:restored", { document: re(), triggerEvent: ce }) }) } else { if (r) { r(e) } } }; setTimeout(function () { ce(e, "htmx:load", {}); e = null }, 0) }); return Q }() });`

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

func createTwColorsFile(projectName string) {
	cssContent := `:root{--tw-slate-50:#f8fafc;--tw-slate-100:#f1f5f9;--tw-slate-200:#e2e8f0;--tw-slate-300:#cbd5e1;--tw-slate-400:#94a3b8;--tw-slate-500:#64748b;--tw-slate-600:#475569;--tw-slate-700:#334155;--tw-slate-800:#1e293b;--tw-slate-900:#0f172a;--tw-gray-50:#f9fafb;--tw-gray-100:#f3f4f6;--tw-gray-200:#e5e7eb;--tw-gray-300:#d1d5db;--tw-gray-400:#9ca3af;--tw-gray-500:#6b7280;--tw-gray-600:#4b5563;--tw-gray-700:#374151;--tw-gray-800:#1f2937;--tw-gray-900:#111827;--tw-zinc-50:#fafafa;--tw-zinc-100:#f4f4f5;--tw-zinc-200:#e4e4e7;--tw-zinc-300:#d4d4d8;--tw-zinc-400:#a1a1aa;--tw-zinc-500:#71717a;--tw-zinc-600:#52525b;--tw-zinc-700:#3f3f46;--tw-zinc-800:#27272a;--tw-zinc-900:#18181b;--tw-neutral-50:#fafafa;--tw-neutral-100:#f5f5f5;--tw-neutral-200:#e5e5e5;--tw-neutral-300:#d4d4d4;--tw-neutral-400:#a3a3a3;--tw-neutral-500:#737373;--tw-neutral-600:#525252;--tw-neutral-700:#404040;--tw-neutral-800:#262626;--tw-neutral-900:#171717;--tw-stone-50:#fafaf9;--tw-stone-100:#f5f5f4;--tw-stone-200:#e7e5e4;--tw-stone-300:#d6d3d1;--tw-stone-400:#a8a29e;--tw-stone-500:#78716c;--tw-stone-600:#57534e;--tw-stone-700:#44403c;--tw-stone-800:#292524;--tw-stone-900:#1c1917;--tw-red-50:#fef2f2;--tw-red-100:#fee2e2;--tw-red-200:#fecaca;--tw-red-300:#fca5a5;--tw-red-400:#f87171;--tw-red-500:#ef4444;--tw-red-600:#dc2626;--tw-red-700:#b91c1c;--tw-red-800:#991b1b;--tw-red-900:#7f1d1d;--tw-orange-50:#fff7ed;--tw-orange-100:#ffedd5;--tw-orange-200:#fed7aa;--tw-orange-300:#fdba74;--tw-orange-400:#fb923c;--tw-orange-500:#f97316;--tw-orange-600:#ea580c;--tw-orange-700:#c2410c;--tw-orange-800:#9a3412;--tw-orange-900:#7c2d12;--tw-amber-50:#fffbeb;--tw-amber-100:#fef3c7;--tw-amber-200:#fde68a;--tw-amber-300:#fcd34d;--tw-amber-400:#fbbf24;--tw-amber-500:#f59e0b;--tw-amber-600:#d97706;--tw-amber-700:#b45309;--tw-amber-800:#92400e;--tw-amber-900:#78350f;--tw-yellow-50:#fefce8;--tw-yellow-100:#fef9c3;--tw-yellow-200:#fef08a;--tw-yellow-300:#fde047;--tw-yellow-400:#facc15;--tw-yellow-500:#eab308;--tw-yellow-600:#ca8a04;--tw-yellow-700:#a16207;--tw-yellow-800:#854d0e;--tw-yellow-900:#713f12;--tw-lime-50:#f7fee7;--tw-lime-100:#ecfccb;--tw-lime-200:#d9f99d;--tw-lime-300:#bef264;--tw-lime-400:#a3e635;--tw-lime-500:#84cc16;--tw-lime-600:#65a30d;--tw-lime-700:#4d7c0f;--tw-lime-800:#3f6212;--tw-lime-900:#365314;--tw-green-50:#f0fdf4;--tw-green-100:#dcfce7;--tw-green-200:#bbf7d0;--tw-green-300:#86efac;--tw-green-400:#4ade80;--tw-green-500:#22c55e;--tw-green-600:#16a34a;--tw-green-700:#15803d;--tw-green-800:#166534;--tw-green-900:#14532d;--tw-emerald-50:#ecfdf5;--tw-emerald-100:#d1fae5;--tw-emerald-200:#a7f3d0;--tw-emerald-300:#6ee7b7;--tw-emerald-400:#34d399;--tw-emerald-500:#10b981;--tw-emerald-600:#059669;--tw-emerald-700:#047857;--tw-emerald-800:#065f46;--tw-emerald-900:#064e3b;--tw-teal-50:#f0fdfa;--tw-teal-100:#ccfbf1;--tw-teal-200:#99f6e4;--tw-teal-300:#5eead4;--tw-teal-400:#2dd4bf;--tw-teal-500:#14b8a6;--tw-teal-600:#0d9488;--tw-teal-700:#0f766e;--tw-teal-800:#115e59;--tw-teal-900:#134e4a;--tw-cyan-50:#ecfeff;--tw-cyan-100:#cffafe;--tw-cyan-200:#a5f3fc;--tw-cyan-300:#67e8f9;--tw-cyan-400:#22d3ee;--tw-cyan-500:#06b6d4;--tw-cyan-600:#0891b2;--tw-cyan-700:#0e7490;--tw-cyan-800:#155e75;--tw-cyan-900:#164e63;--tw-sky-50:#f0f9ff;--tw-sky-100:#e0f2fe;--tw-sky-200:#bae6fd;--tw-sky-300:#7dd3fc;--tw-sky-400:#38bdf8;--tw-sky-500:#0ea5e9;--tw-sky-600:#0284c7;--tw-sky-700:#0369a1;--tw-sky-800:#075985;--tw-sky-900:#0c4a6e;--tw-blue-50:#eff6ff;--tw-blue-100:#dbeafe;--tw-blue-200:#bfdbfe;--tw-blue-300:#93c5fd;--tw-blue-400:#60a5fa;--tw-blue-500:#3b82f6;--tw-blue-600:#2563eb;--tw-blue-700:#1d4ed8;--tw-blue-800:#1e40af;--tw-blue-900:#1e3a8a;--tw-indigo-50:#eef2ff;--tw-indigo-100:#e0e7ff;--tw-indigo-200:#c7d2fe;--tw-indigo-300:#a5b4fc;--tw-indigo-400:#818cf8;--tw-indigo-500:#6366f1;--tw-indigo-600:#4f46e5;--tw-indigo-700:#4338ca;--tw-indigo-800:#3730a3;--tw-indigo-900:#312e81;--tw-violet-50:#f5f3ff;--tw-violet-100:#ede9fe;--tw-violet-200:#ddd6fe;--tw-violet-300:#c4b5fd;--tw-violet-400:#a78bfa;--tw-violet-500:#8b5cf6;--tw-violet-600:#7c3aed;--tw-violet-700:#6d28d9;--tw-violet-800:#5b21b6;--tw-violet-900:#4c1d95;--tw-purple-50:#faf5ff;--tw-purple-100:#f3e8ff;--tw-purple-200:#e9d5ff;--tw-purple-300:#d8b4fe;--tw-purple-400:#c084fc;--tw-purple-500:#a855f7;--tw-purple-600:#9333ea;--tw-purple-700:#7e22ce;--tw-purple-800:#6b21a8;--tw-purple-900:#581c87;--tw-fuchsia-50:#fdf4ff;--tw-fuchsia-100:#fae8ff;--tw-fuchsia-200:#f5d0fe;--tw-fuchsia-300:#f0abfc;--tw-fuchsia-400:#e879f9;--tw-fuchsia-500:#d946ef;--tw-fuchsia-600:#c026d3;--tw-fuchsia-700:#a21caf;--tw-fuchsia-800:#86198f;--tw-fuchsia-900:#701a75;--tw-pink-50:#fdf2f8;--tw-pink-100:#fce7f3;--tw-pink-200:#fbcfe8;--tw-pink-300:#f9a8d4;--tw-pink-400:#f472b6;--tw-pink-500:#ec4899;--tw-pink-600:#db2777;--tw-pink-700:#be185d;--tw-pink-800:#9d174d;--tw-pink-900:#831843;--tw-rose-50:#fff1f2;--tw-rose-100:#ffe4e6;--tw-rose-200:#fecdd3;--tw-rose-300:#fda4af;--tw-rose-400:#fb7185;--tw-rose-500:#f43f5e;--tw-rose-600:#e11d48;--tw-rose-700:#be123c;--tw-rose-800:#9f1239;--tw-rose-900:#881337}`

	filePath := filepath.Join(projectName, "static", "twcolors.min.css")

	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println("error creating twcolors.min.css file: ", err)
	}
	defer f.Close()

	_, err = f.WriteString(cssContent)
	if err != nil {
		fmt.Println("error writing twcolors.min.css content to file: ", err)
	}
}

func createCssFile(projectName string) {
	cssContent := `* {
  padding: 0;
  margin: 0;
  box-sizing: border-box;
}

html,
body {
  height: 100%;
  width: 100%;
  font-family: Verdana, Arial, Tahoma, sans-serif;
  font-size: 1rem;
}

main {
  height: 100%;
  width: 100%;
}

.btn {
  border: none;
  border-radius: 0.25rem;
  padding: 0.5rem 1rem;
  cursor: pointer;
  background: var(--tw-green-500);
  color: var(--tw-slate-100);
  font-weight: bold;
}

.btn-ghost {
  padding: 0;
  border: none;
  cursor: pointer;
  color: var(--tw-indigo-500);
  background: none;
  font-weight: bold;
  font-size: 1rem;
}

.container {
  margin: 0 auto;
  padding: 0 1rem;
  max-width: 1440px;
}

.nav {
  padding: 1.5rem 0;
  position: fixed;
  width: 100%;
  box-shadow: 0 25px 50px -12px rgb(0 0 0 / 0.25);
}

.nav__content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.nav__brand {
  display: flex;
  align-items: center;
  color: var(--tw-slate-100);
  font-size: 1.25rem;
  font-weight: bold;
  text-decoration: none;
  text-transform: uppercase;
}

.nav__brand>svg {
  height: 1.8rem;
  width: 1.8rem;
}

.nav__list {
  width: 325px;
  display: flex;
  list-style: none;
  justify-content: flex-end;
}

.nav__item {
  margin-right: 1rem;
  min-width: fit-content;
}

.nav__item:last-child {
  margin: 0;
}

.nav__link {
  cursor: pointer;
  color: var(--tw-slate-100);
  text-decoration: none;
  text-transform: uppercase;
  background: none;
  border: none;
  font-size: 1rem;
}

.nav__link:hover {
  border-bottom: 2px solid var(--tw-slate-100);
}

.hero {
  padding: 1.5rem;
  height: 100%;
  width: 100%;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  color: var(--tw-slate-100);
  background: linear-gradient(
    to right,
	var(--tw-indigo-700),
	var(--tw-blue-600),
	var(--tw-indigo-700));
  );
}

.hero__title {
  padding-bottom: 2rem;
  font-size: 1.75rem;
}

.hero__intro {
  padding-bottom: 2rem;
  font-size: 1.2rem;
}

.waitlist-form {
  display: flex;
  flex-direction: column;
  min-width: 22rem;
}

.waitlist-form__group {
  padding-top: 1rem;
}

.waitlist-form__label {
}

.waitlist-form__input {
  padding: 0.5rem;
  width: 100%;
  margin-bottom: 1rem;
  border: none;
  border-radius: 0.25rem;
}

.waitlist-form__message {
  height: 2.25rem;
  padding-top: 1rem;
  padding-bottom: 0;
  text-align: center;
  font-size: 1rem;
}

.waitlist-form__message-error {
  color: var(--tw-red-500);
}

.waitlist-form__btn {
  width: 100%;
}

.waitlist-form__message,
.waitlist-form__message-error {
  text-align: center;
  margin-top: 0.5rem;
}

.auth-form__wrapper {
  padding: 0 1rem;
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
}

.auth-form {
  padding: 2.75rem;
  display: flex;
  flex-direction: column;
  border: solid 2px var(--tw-slate-900);
  border-radius: 0.25rem;
}

.auth-form__title {
  display: flex;
  justify-content: center;
  align-items: center;
  color: var(--tw-indigo-600);
  font-size: 1.75rem;
  font-weight: bold;
  text-decoration: none;
  text-transform: uppercase;
  margin-bottom: 1rem;
}

.auth-form__title>svg {
  height: 2.8rem;
  width: 2.8rem;
}

.auth-form__group {
  padding-top: 0.5rem;
}

.auth-form__label {
  margin-bottom: 0.25rem;
  color: var(--tw-slate-900);
}

.auth-form__input {
  font-size: 1.1rem;
  padding: 0.5rem;
  width: 100%;
  margin: 0.5rem 0;
  border: solid 1px var(--tw-slate-900);
  border-radius: 0.25rem;
}

.auth-form__message {
  height: 2.25rem;
  padding-top: 1rem;
  padding-bottom: 0;
  text-align: center;
  font-size: 1rem;
}

.auth-form__message-error {
  color: var(--tw-red-500);
}

.auth-form__btn {
  margin-top: 1rem;
  padding: 0.75rem;
  width: 100%;
  background: var(--tw-indigo-600);
}

.auth-form__message {
  height: 2.25rem;
  padding-top: 1rem;
  padding-bottom: 0;
  text-align: center;
  font-size: 1rem;
}

.auth-form__message-error {
  color: var(--tw-red-500);
}

.auth-form__btn {
  width: 100%;
}

.auth-form__type {
  text-align: center;
  margin-top: 2rem;
}

.dashboard__wrapper {
  width: 100vw;
  height: 100vh;
  display: flex;
}

.dashboard__navigation {
  width: 100%;
  max-width: 300px;
  height: 100%;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  color: var(--tw-slate-100);
  background: linear-gradient(to bottom,
      var(--tw-indigo-700),
      var(--tw-blue-600),
      var(--tw-indigo-700));
  border-right: 1px solid var(--tw-slate-900);
}

.dashboard__branding {
  padding: 1rem;
}

.dashboard__navigation-list,
.dashboard__navigation-admin-list {
  padding: 1.5rem 1rem;
}

.dashboard__branding {
  font-size: 1.5rem;
  text-align: center;
  box-shadow: 0 25px 50px -12px rgb(0 0 0 / 0.25);
}

.dashboard__navigation-list {
  list-style: none;
}

.dashboard__navigation-admin-separator {
  border-top: 1px solid var(--tw-slate-100);
  padding: 0 1rem;
  width: 90%;
  margin: 0 auto;
  opacity: 0.5;
}

.dashboard__navigation-admin-list {
  list-style: none;
}

.dashboard__navigation-item {
  margin: 0.75rem 0;
}

.dashboard__navigation-item:first-child {
  margin-top: 0;
  margin-bottom: 0.75rem;
}

.dashboard__navigation-link {
  color: var(--tw-slate-100);
  background: none;
  border: none;
  font-size: 1.15rem;
  cursor: pointer;
  display: flex;
  align-items: center;
}

.dashboard__navigation-link>svg {
  height: 1.5rem;
  width: 1.5rem;
  margin-right: 0.75rem;
}

.dashboard__navigation-sign-out {
  margin: 1rem;
  padding: 0.5rem 1rem;
  background: var(--tw-slate-100);
  color: var(--tw-slate-900);
  font-size: 1rem;
}

.dashboard__content {
  padding: 1.5rem 1rem;
  width: 100%;
  height: 100%;
  background: var(--tw-slate-100);
}

@media screen and (min-width: 768px) {
.nav__brand {
    font-size: 1.5rem;
  }

  .hero__title {
    font-size: 4.25rem;
  }

  .hero__intro {
    font-size: 1.5rem;
    text-align: center;
    width: 100%;
    max-width: 68ch;
  }

  .waitlist-form {
    min-width: 32rem;
  }

  .waitlist-form__group {
    flex-direction: row;
    display: flex;
    align-items: center;
  }

  .waitlist-form__label {
    min-width: 24rem;
  }

  .waitlist-form__input {
    margin-bottom: 0;
    width: 95%;
  }

  .auth-form__title {
    font-size: 2.5rem;
  }
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
	dbFilename := strings.ToLower(projectName) + ".db"

	ignoreContent := fmt.Sprintf(
		`# Created by https://www.toptal.com/developers/gitignore/api/go,linux,windows,macos
# Edit at https://www.toptal.com/developers/gitignore?templates=go,linux,windows,macos

.env
bin
%s

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
`,
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

	dotenvContent := fmt.Sprintf(`%s_DB_PATH="%s"
	%s_COOKIE_STORE_SECRET="%s"
	`, dbEnv, dbFilename, sessEnv, sessSecret)

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

func createMakefile(projectName string) {
	makefileContent := fmt.Sprintf(`BINARY_NAME=%s

all: clean format build run

build:
	@echo "Building..."
	@go build -o bin/$(BINARY_NAME) cmd/main.go

clean:
	@echo "Cleaning..."
	@go clean
	@if [ -e bin/$(BINARY_NAME) ]; then rm bin/$(BINARY_NAME); fi

format:
	@echo "Formatting..."
	@go fmt ./...

run:
	@echo "Running..."
	@./bin/$(BINARY_NAME)

test:
	@echo "Testing..."
	@go test ./...
`, projectName)

	filePath := filepath.Join(projectName, "Makefile")

	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println("error creating Makefile file: ", err)
	}
	defer f.Close()

	_, err = f.WriteString(makefileContent)
	if err != nil {
		fmt.Println("error writing Makefile content to file: ", err)
	}
}

func createDockerfile(projectName string) {
	dockerfileContent := fmt.Sprintf(`# Use the official Ubuntu image as the base image
FROM ubuntu:latest

# Set the working directory inside the container
WORKDIR /app

# Copy pre-built binary into the container
COPY bin/%s /app/%s

# Copy all static files into the container
COPY static /app/static

# Copy all template files into the container
COPY template /app/template

# Expose port 8080 to run the application
EXPOSE 8080

# Command to run the application
CMD ["./%s"]
`, projectName, projectName, projectName)

	filePath := filepath.Join(projectName, "Dockerfile")

	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println("error creating Dockerfile file: ", err)
	}
	defer f.Close()

	_, err = f.WriteString(dockerfileContent)
	if err != nil {
		fmt.Println("error writing Dockerfile content to file: ", err)
	}
}
