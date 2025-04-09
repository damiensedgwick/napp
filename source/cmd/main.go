package main

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
	db.AutoMigrate(&User{})

	// Pages
	e.GET("/", homePageHandler())
	e.GET("/dashboard", dashboardPageHandler())

	// Blocks
	e.GET("/auth/sign-in", signIn())
	e.POST("/auth/sign-in", signInWithEmailAndPassword(db))
	e.GET("/auth/sign-up", signUp())
	e.POST("/auth/sign-up", signUpWithEmailAndPassword(db))
	e.POST("/auth/sign-out", signOut())

	e.Logger.Fatal(e.Start(":8080"))
}

type HomePageData struct {
	User User
}

func newPageData(user User) HomePageData {
	return HomePageData{
		User: user,
	}
}

func homePageHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, _ := session.Get("session", c)
		if sess.Values["user"] != nil {
			var user User
			err := json.Unmarshal(sess.Values["user"].([]byte), &user)
			if err != nil {
				fmt.Println("error unmarshalling user value")
				return err
			}

			return c.Render(200, "index", newPageData(user))
		}

		return c.Render(200, "index", nil)
	}
}

type DashboardPageData struct {
	User User
}

func newDashboardData(user User) DashboardPageData {
	return DashboardPageData{
		User: user,
	}
}

func dashboardPageHandler() echo.HandlerFunc {
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
		if compareErr := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); compareErr != nil {
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
