package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	files "./libs/files"

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
	// setup the volume for the server.
	server.volume = files.Volume{Name: "C:", Path: "./files"}
	// Setup fiber
	app := fiber.New()
	app.Settings.TemplateEngine = template.Handlebars()
	// setup logger middleware
	app.Use(logger.New())
	// setup static routes
	app.Static("/css", "./styles/css")
	app.Static("/js", "./js")
	app.Static("/media", "./media")
	app.Static("/volume", server.volume.Path)
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
	app.Get("/home", server.home)
	app.Get("/pdf-viewer", server.pdfViewer)
	app.Post("/pdf-update", server.pdfUpdate)
	// start the server on the server.port
	log.Fatal(app.Listen(server.port))
}

// < ----- User ----- >
type User struct {
	ID             string `json: "ID" db: "ID"`
	ProfilePicture string `json: "profilepicture" db: "ProfilePicture"`
	Password       string `json: "password" db: "Password"`
	Username       string `json: "password" db: "Username"`
}

type PDF struct {
	ID   string `json: "ID" db: "ID"`
	Hash string `json: "Hash" db: "Hash"`
	Path string `json: "Path" db: "Path"`
	Page int    `json: "Page" db: "Page"`
}

// < ----- Server ----- >

// Server class
type Server struct {
	db     *sql.DB
	secret string
	port   int
	volume files.Volume
}

// Signup
func (server *Server) signup(c *fiber.Ctx) {
	user := &User{Username: c.FormValue("username"), Password: c.FormValue("password"), ProfilePicture: c.FormValue("profilepicture")}
	if len(user.Username) < 3 || len(user.Password) < 6 {
		c.SendStatus(fiber.StatusBadRequest)
		return
	}
	userExists := server.getUserByUsername(user.Username)
	if userExists.Username == user.Username {
		c.SendStatus(fiber.StatusForbidden)
		return
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 8)
	err = server.insertUser(user.Username, string(hashedPassword), user.ProfilePicture)
	if err != nil {
		c.SendStatus(fiber.StatusInternalServerError)
		fmt.Println(err.Error())
		return
	}
	c.Redirect("/signin")
}

// Signin
func (server *Server) signin(c *fiber.Ctx) {
	user := &User{Username: c.FormValue("username"), Password: c.FormValue("password"), ProfilePicture: c.FormValue("profilepicture")}
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
	c.Redirect("/home?path=/")
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

// home
func (server *Server) home(c *fiber.Ctx) {
	//user := c.Locals("user").(*jwt.Token)
	//claims := user.Claims.(jwt.MapClaims)
	fmt.Println("Query path length: ", len(c.Query("path")))
	qPath := server.volume.Path
	if len(c.Query("path")) == 0 {
		qPath += "/"
	} else {
		qPath += c.Query("path")
	}
	fmt.Println("Query path: ", qPath)
	files, err := server.volume.WalkFolder(qPath)
	if err != nil {
		fmt.Println(err.Error())
	}
	path := strings.Split(qPath, "./")
	path = strings.Split(path[1], "/")
	path = delete_empty(path)
	if err != nil {
		c.Status(500).Send(err.Error())
	}
	endpath := ""
	volumepath := ""
	if len(path) < 2 {
		endpath = server.volume.Name
		path = nil
	} else {
		volumepath = server.volume.Name
		endpath = path[len(path)-1]
		path = path[1 : len(path)-1]
	}
	bind := fiber.Map{
		"username": "claims[\"name\"].(string)",
		"file":     files, "volumepath": volumepath,
		"path":    path,
		"endpath": endpath,
	}
	if err = c.Render("./views/home.handlebars", bind); err != nil {
		c.Status(500).Send(err.Error())
	}
}

// pdf-viewer
func (server *Server) pdfViewer(c *fiber.Ctx) {
	Hash := c.Query("hash")
	pdf := server.getPdfByHash(Hash)
	temp := PDF{}
	if pdf.ID == temp.ID {
		pdf.Hash = Hash
		pdf.Path = c.Query("path")
		pdf.Page = 1
		err := server.insertPdf(pdf.Hash, pdf.Path, pdf.Page)
		if err != nil {
			fmt.Println(err.Error())
			c.SendStatus(fiber.StatusBadRequest)
		}
	}
	bind := fiber.Map{"config": true, "Page": pdf.Page, "Hash": pdf.Hash, "Path": pdf.Path}
	if err := c.Render("./views/pdf-viewer.handlebars", bind); err != nil {
		c.Status(500).Send(err.Error())
	}
}

// update pdf
func (server *Server) pdfUpdate(c *fiber.Ctx) {
	fmt.Println("Page:", c.Query("page"))
	Page, err := strconv.Atoi(c.Query("page"))
	if err != nil {
		c.SendStatus(fiber.StatusBadRequest)
	}
	Hash := c.Query("hash")
	Path := c.Query("path")
	err = server.updatePdfPageCount(Hash, Path, Page)
	if err != nil {
		c.SendStatus(fiber.StatusBadRequest)
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
	server.db, err = sql.Open("sqlite3", "./database.db")
	if err != nil {
		panic(err)
	}
	// Setup the database table if it doesn't exist'
	statement, err := server.db.Prepare(`
		CREATE TABLE IF NOT EXISTS users(
			ID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			Username TEXT,
			Password TEXT,
			ProfilePicture TEXT
		);
	`)
	if err != nil {
		panic(err)
	}
	statement.Exec()
	statement, err = server.db.Prepare(`
		CREATE TABLE IF NOT EXISTS pdfs(
			ID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			Hash TEXT,
			Path TEXT,
			Page INTEGER
		);
	`)
	if err != nil {
		panic(err)
	}
	statement.Exec()
}

// insertUser
func (server *Server) insertUser(username string, password string, profilepicture string) error {
	statement, _ := server.db.Prepare(`
		INSERT INTO users (Username, Password, ProfilePicture) values (?,?,?)
	`)
	_, err := statement.Exec(username, password, profilepicture)
	if err != nil {
		return err
	}
	return nil
}

// getUserByUsername
func (server *Server) getUserByUsername(username string) User {
	result := server.db.QueryRow("select * from users where Username=$1", username)
	user := User{}
	result.Scan(&user.ID, &user.Username, &user.Password, &user.ProfilePicture)
	return user
}

// insertPdf
func (server *Server) insertPdf(hash string, path string, page int) error {
	statement, _ := server.db.Prepare(`
		INSERT INTO pdfs (Hash, Path, Page) values (?,?,?)
	`)
	_, err := statement.Exec(hash, path, page)
	if err != nil {
		return err
	}
	return nil
}

// updatePdfPageCount
func (server *Server) updatePdfPageCount(hash string, path string, page int) error {
	statement, _ := server.db.Prepare(`
		UPDATE pdfs SET Page=$1 WHERE Hash=$2
	`)
	result, err := statement.Exec(page, hash)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		err := server.insertPdf(hash, path, page)
		if err != nil {
			return err
		}
	}
	return nil
}

// getPdfByHash
func (server *Server) getPdfByHash(hash string) PDF {
	result := server.db.QueryRow("select * from pdfs where Hash=$1", hash)
	pdf := PDF{}
	result.Scan(&pdf.ID, &pdf.Hash, &pdf.Path, &pdf.Page)
	return pdf
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

// < ----- Helpers ----- >
func delete_empty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}
