package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	files "../files"
	"github.com/dgrijalva/jwt-go"
	"github.com/gofiber/fiber"
	"golang.org/x/crypto/bcrypt"
)

// < ----- User ----- >

// User is a struct representing a user in the database
type User struct {
	ID             string `json: "ID" db: "ID"`
	ProfilePicture string `json: "ProfilePicture" db: "ProfilePicture"`
	Password       string `json: "Password" db: "Password"`
	Username       string `json: "Username" db: "Username"`
}

// < ----- PDF ----- >

// PDF is a struct representing a pdf in the database
type PDF struct {
	ID   string `json: "ID" db: "ID"`
	User string `json: "User" db: "User"`
	Hash string `json: "Hash" db: "Hash"`
	Path string `json: "Path" db: "Path"`
	Page int    `json: "Page" db: "Page"`
}

// < ----- Server ----- >

// Server class
type Server struct {
	DB     *sql.DB
	Secret string
	Port   int
	Volume files.Volume
}

// Signup is the path used for createing a user. the username need to be  unique.
func (server *Server) Signup(c *fiber.Ctx) {
	user := &User{Username: c.FormValue("username"), Password: c.FormValue("password"), ProfilePicture: c.FormValue("profilepicture")}
	if len(user.Username) < 3 || len(user.Password) < 6 {
		c.SendStatus(fiber.StatusBadRequest)
		return
	}
	userExists := server.GetUserByUsername(user.Username)
	if userExists.Username == user.Username {
		c.SendStatus(fiber.StatusForbidden)
		return
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 8)
	err = server.InsertUser(user.Username, string(hashedPassword), user.ProfilePicture)
	if err != nil {
		c.SendStatus(fiber.StatusInternalServerError)
		fmt.Println(err.Error())
		return
	}
	c.Redirect("/signin")
}

// Signin is used to assign the user their token given they provided the correct credentials.
func (server *Server) Signin(c *fiber.Ctx) {
	user := &User{Username: c.FormValue("username"), Password: c.FormValue("password")}
	storedUser := server.GetUserByUsername(user.Username)
	if storedUser.ID == sql.ErrNoRows.Error() {
		c.SendStatus(http.StatusUnauthorized)
		return
	}
	err := bcrypt.CompareHashAndPassword([]byte(storedUser.Password), []byte(user.Password))
	if err != nil {
		c.SendStatus(fiber.StatusUnauthorized)
		return
	}
	server.generateJWTToken(c, user.Username, storedUser.ProfilePicture)
	c.Redirect("/home?path=/")
}

// Login is the frontend used to both signin and signup.
func (server *Server) Login(c *fiber.Ctx) {
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

// Home shows all the current files in the given directory.
func (server *Server) Home(c *fiber.Ctx) {
	// Get current user information from the claims map.
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)

	qPath := server.Volume.Path
	if len(c.Query("path")) == 0 {
		qPath += "/"
	} else {
		qPath += c.Query("path")
	}

	files, err := server.Volume.WalkFolder(qPath)
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
	Volumepath := ""
	if len(path) < 2 {
		endpath = server.Volume.Name
		path = nil
	} else {
		Volumepath = server.Volume.Name
		endpath = path[len(path)-1]
		path = path[1 : len(path)-1]
	}
	bind := fiber.Map{
		"username":       claims["username"].(string),
		"profilepicture": claims["profilepicture"].(string),
		"file":           files, "volumepath": Volumepath,
		"path":    path,
		"endpath": endpath,
	}
	if err = c.Render("./views/home.handlebars", bind); err != nil {
		c.Status(500).Send(err.Error())
	}
}

// Pdf-viewer servers the page that renders the pdf.
func (server *Server) PdfViewer(c *fiber.Ctx) {
	// Get current user information from the claims map.
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)

	Hash := c.Query("hash")
	pdf := server.GetPdfByHash(claims["username"].(string), Hash)
	temp := PDF{}
	if pdf.ID == temp.ID {
		pdf.Hash = Hash
		pdf.Path = c.Query("path")
		pdf.Page = 1
		pdf.User = claims["username"].(string)
		err := server.InsertPdf(pdf.User, pdf.Hash, pdf.Path, pdf.Page)
		if err != nil {
			fmt.Println(err.Error())
			c.SendStatus(fiber.StatusBadRequest)
		}
	}
	bind := fiber.Map{
		"config":         true,
		"Page":           pdf.Page,
		"Hash":           pdf.Hash,
		"Path":           pdf.Path,
		"username":       claims["username"].(string),
		"profilepicture": claims["profilepicture"].(string),
	}
	if err := c.Render("./views/pdf-viewer.handlebars", bind); err != nil {
		c.Status(500).Send(err.Error())
	}
}

// PdfUpdate updates the pdf information in the database.
func (server *Server) PdfUpdate(c *fiber.Ctx) {
	Page, err := strconv.Atoi(c.Query("page"))
	if err != nil {
		c.SendStatus(fiber.StatusBadRequest)
	}
	User := c.Query("user")
	Hash := c.Query("hash")
	Path := c.Query("path")
	err = server.UpdatePdfPageCount(User, Hash, Path, Page)
	if err != nil {
		fmt.Println(err.Error())
		c.SendStatus(fiber.StatusBadRequest)
	}
}

// Generate JWT token
func (server *Server) generateJWTToken(c *fiber.Ctx, username string, profilepicture string) {
	// Create token
	token := jwt.New(jwt.SigningMethodHS256)

	// Set claims
	claims := token.Claims.(jwt.MapClaims)
	claims["username"] = username
	claims["profilepicture"] = profilepicture
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

	// Generate encoded token and send it as response.
	genToken, err := token.SignedString([]byte(server.Secret))
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
func (server *Server) JwtErrorHandler(c *fiber.Ctx, err error) {
	fmt.Println("err:", err.Error())
	c.Redirect("/signin", 302)
}

// < ----- DATABASE ----- >

// InitDB initializes the database.
func (server *Server) InitDB() {
	// Connect to the postgres db
	//you might have to change the connection string to add your database credentials
	var err error
	server.DB, err = sql.Open("sqlite3", "./db/database.db")
	if err != nil {
		panic(err)
	}
	// Setup the database table if it doesn't exist'
	statement, err := server.DB.Prepare(`
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
	statement, err = server.DB.Prepare(`
		CREATE TABLE IF NOT EXISTS pdfs(
			ID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			User TEXT,
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

// InsertUser inserts a user into the database.
func (server *Server) InsertUser(username string, password string, profilepicture string) error {
	statement, _ := server.DB.Prepare(`
		INSERT INTO users (Username, Password, ProfilePicture) values (?,?,?)
	`)
	_, err := statement.Exec(username, password, profilepicture)
	if err != nil {
		return err
	}
	return nil
}

// GetUserByUsername gets the user by their username and returns the user as a User object.
func (server *Server) GetUserByUsername(username string) User {
	result := server.DB.QueryRow("select * from users where Username=$1", username)
	user := User{}
	result.Scan(&user.ID, &user.Username, &user.Password, &user.ProfilePicture)
	return user
}

// InsertPdf inserts a pdf into the database
func (server *Server) InsertPdf(user string, hash string, path string, page int) error {
	statement, _ := server.DB.Prepare(`
		INSERT INTO pdfs (User, Hash, Path, Page) values (?,?,?,?)
	`)
	_, err := statement.Exec(user, hash, path, page)
	if err != nil {
		return err
	}
	return nil
}

// UpdatePdfPageCount updates the current page count of a given pdf.
func (server *Server) UpdatePdfPageCount(user string, hash string, path string, page int) error {
	statement, _ := server.DB.Prepare(`
		UPDATE pdfs SET Page=$1 WHERE Hash=$2 AND User=$3
	`)
	result, err := statement.Exec(page, hash, user)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		err := server.InsertPdf(user, hash, path, page)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetPdfByHash gets the pdf from it's hash and returns it as a PDF object.
func (server *Server) GetPdfByHash(user string, hash string) PDF {
	result := server.DB.QueryRow("SELECT * FROM pdfs WHERE User=$1 AND Hash=$2", user, hash)
	pdf := PDF{}
	result.Scan(&pdf.ID, &pdf.User, &pdf.Hash, &pdf.Path, &pdf.Page)
	return pdf
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
