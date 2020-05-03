package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gofiber/fiber"
	jwtware "github.com/gofiber/jwt"
	"github.com/gofiber/logger"
	"github.com/gofiber/template"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Setup the server
	server := &Server{}
	flags(server)
	// Setup the database
	server.initDB()
	// Setup fiber
	app := fiber.New()
	app.Settings.TemplateEngine = template.Handlebars()
	// setup logger middleware
	app.Use(logger.New())
	// setup static routes
	app.Static("/css", "./styles/css")
	app.Static("/js", "./js")
	app.Static("/media", "./media")
	// setup GET routes
	app.Get("/signin", server.login)
	app.Get("/signup", server.login)
	// setup POST routes
	app.Post("/signin", server.signin)
	app.Post("/signup", server.signup)
	// setup authentication routes
	app.Use(jwtware.New(jwtware.Config{
		SigningKey:   []byte(server.secret),
		TokenLookup:  "cookie:token",
		ErrorHandler: server.jwtErrorHandler,
	}))
	// setup GET routes
	app.Get("/secret", func(c *fiber.Ctx) {
		user := c.Locals("user").(*jwt.Token)
		claims := user.Claims.(jwt.MapClaims)
		name := claims["name"].(string)
		c.Send("Welcome " + name)
	})
	app.Get("/home", server.session)
	// start the server on the server.port
	log.Fatal(app.Listen(server.port))
}

// < ----- User ----- >
type User struct {
	ID       string `json: "ID" db: "ID"`
	Password string `json: "password" db: "password"`
	Username string `json: "password" db: "username"`
}

// < ----- Server ----- >

// Server class
type Server struct {
	db     *sql.DB
	secret string
	port   int
}

// Signup
func (server *Server) signup(c *fiber.Ctx) {
	user := &User{Username: c.FormValue("username"), Password: c.FormValue("password")}
	fmt.Println("User: ", user)
	if len(user.Username) < 3 || len(user.Password) < 6 {
		c.SendStatus(fiber.StatusBadRequest)
		return
	}
	userExists := server.getUserByUsername(user.Username)
	if userExists.Username == user.Username {
		c.SendStatus(fiber.StatusForbidden)
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 8)
	fmt.Println("hashedPassword: ", string(hashedPassword))
	err = server.insertUser(user.Username, string(hashedPassword))
	if err != nil {
		c.SendStatus(fiber.StatusInternalServerError)
		return
	}
	c.SendStatus(fiber.StatusOK)
}

// Signin
func (server *Server) signin(c *fiber.Ctx) {
	user := &User{Username: c.FormValue("username"), Password: c.FormValue("password")}
	storedUser := server.getUserByUsername(user.Username)
	if storedUser.ID == sql.ErrNoRows.Error() {
		c.SendStatus(http.StatusUnauthorized)
		return
	}
	err := bcrypt.CompareHashAndPassword([]byte(storedUser.Password), []byte(user.Password))
	if err != nil {
		c.SendStatus(fiber.StatusUnauthorized)
	}
	server.generateJWTToken(c, user.Username)
}

// Login
func (server *Server) login(c *fiber.Ctx) {
	if c.Path() == "/signup" {
		if err := c.Render("./views/login.handlebars", fiber.Map{"signup": true}); err != nil {
			c.Status(500).Send(err.Error())
		}
	} else {
		if err := c.Render("./views/login.handlebars", fiber.Map{"signup": false}); err != nil {
			c.Status(500).Send(err.Error())
		}
	}
}

// session.
func (server *Server) session(c *fiber.Ctx) {
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	_user := fiber.Map{
		"name": claims["name"].(string),
	}
	if err := c.Render("./views/session.handlebars", _user); err != nil {
		c.Status(500).Send(err.Error())
	}
}

// Generate JWT token
func (server *Server) generateJWTToken(c *fiber.Ctx, username string) {
	// Create token
	token := jwt.New(jwt.SigningMethodHS256)

	// Set claims
	claims := token.Claims.(jwt.MapClaims)
	claims["name"] = username
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

	// Generate encoded token and send it as response.
	genToken, err := token.SignedString([]byte(server.secret))
	if err != nil {
		c.SendStatus(fiber.StatusInternalServerError)
		return
	}
	cookie := new(fiber.Cookie)
	cookie.Name = "token"
	cookie.Value = genToken
	cookie.Expires = time.Now().Add(time.Hour * 72)
	c.Cookie(cookie)
	c.Redirect("/session")
}

// JWT error handler
func (server *Server) jwtErrorHandler(c *fiber.Ctx, err error) {
	fmt.Println("err:", err.Error())
	c.SendStatus(fiber.StatusBadRequest)
}

// < ----- DATABASE ----- >

// initDB
func (server *Server) initDB() {
	// Connect to the postgres db
	//you might have to change the connection string to add your database credentials
	var err error
	server.db, err = sql.Open("sqlite3", "./Users.db")
	if err != nil {
		panic(err)
	}
	// Setup the database table if it doesn't exist'
	statement, err := server.db.Prepare(`
		CREATE TABLE IF NOT EXISTS users(
			ID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			Username TEXT,
			Password TEXT
		);
	`)
	if err != nil {
		panic(err)
	}
	statement.Exec()
}

// insertUser
func (server *Server) insertUser(username string, password string) error {
	statement, _ := server.db.Prepare(`
		INSERT INTO users (username, password) values (?,?)
	`)
	_, err := statement.Exec(username, password)
	if err != nil {
		return err
	}
	return nil
}

// getUserByUsername
func (server *Server) getUserByUsername(username string) User {
	result := server.db.QueryRow("select * from users where username=$1", username)
	user := User{}
	result.Scan(&user.ID, &user.Username, &user.Password)
	return user
}

// < ----- FLAGS ----- >

func flags(server *Server) {
	secret := flag.String("secret", stringWithCharset(256, charset), "The secrect is a key used for signing the users JWT token")
	port := flag.Int("port", 8080, "The port is the port used to server the server")
	flag.Parse()
	server.secret = *secret
	server.port = *port
}

// < ----- Random Generators ----- >
const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func stringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
