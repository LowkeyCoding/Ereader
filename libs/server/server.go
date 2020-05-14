package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	ExtensionAPI "../extension"
	Files "../files"
	"github.com/dgrijalva/jwt-go"
	"github.com/gofiber/fiber"
	"golang.org/x/crypto/bcrypt"
)

// < ----- User ----- >

// User is a struct representing a user in the database
type User struct {
	ID             string             `json:"ID"`
	Username       string             `json:"Username"`
	Password       string             `json:"Password"`
	ProfilePicture string             `json:"ProfilePicture"`
	FileSettings   Files.FileSettings `json:"FileSettings"`
}

// < ----- Server ----- >

// Server class
type Server struct {
	DB        *sql.DB
	Username  string
	Password  string
	Secret    string
	Port      int
	Etag      bool
	Volume    Files.Volume
	IconsList map[string]bool
}

// < ----- POST ROUTES ----- >

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

// UpdateSetting is used to either update or create a setting.
func (server *Server) UpdateSetting(c *fiber.Ctx) {
	fileSetting := &Files.FileSetting{Username: c.FormValue("Username"), Extension: c.FormValue("Extension"), ApplicationLink: c.FormValue("ApplicationLink")}
	if fileSetting.Extension[0] != '.' {
		c.SendStatus(fiber.StatusBadRequest)
		return
	}
	err := server.UpdateFileSetting(fileSetting.Username, fileSetting.Extension, fileSetting.ApplicationLink)
	if err != nil {
		c.SendStatus(fiber.StatusInternalServerError)
		fmt.Println(err.Error())
		return
	}
	c.SendStatus(fiber.StatusOK)
}

// Query is the path used by extension to query their database table.
func (server *Server) Query(c *fiber.Ctx) {
	var databaseQuery ExtensionAPI.DatabaseQuery

	databaseQuery.TableName = c.FormValue("Result")

	var VariableType map[string]ExtensionAPI.DatabaseItemType
	json.Unmarshal([]byte(c.FormValue("VariableType")), &VariableType)
	databaseQuery.VariableType = VariableType

	var Contains map[string]string
	json.Unmarshal([]byte(c.FormValue("Contains")), &Contains)
	databaseQuery.Contains = Contains

	var Set map[string]string
	json.Unmarshal([]byte(c.FormValue("Set")), &Set)
	databaseQuery.Set = Set

	databaseQuery.TableName = c.FormValue("TableName")

	databaseQuery.DatabaseOperation = ExtensionAPI.DatabaseOperationType(c.FormValue("DatabaseOperation"))

	result, err := databaseQuery.GenerateQuery(server.DB)
	if err != nil {
		fmt.Println(err.Error())
	}
	c.SendString(result)
}

// GetFiles is used to retrieve all files from a given path.
func (server *Server) GetFiles(c *fiber.Ctx) {
	// Get current user information from the claims map.
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)

	qPath := server.Volume.Path
	if len(c.Query("path")) == 0 {
		qPath += "/"
	} else {
		qPath += c.Query("path")
	}
	Files, err := server.Volume.WalkFolder(qPath)
	tUser := server.GetUserByUsername(claims["username"].(string))

	settingsMap := tUser.FileSettings.ToMap()
	Files = Files.AddFileSetting(settingsMap, server.IconsList)

	if err != nil {
		fmt.Println(err.Error())
		c.SendStatus(fiber.StatusInternalServerError)
	}
	json, err := json.Marshal(Files)
	if err != nil {
		fmt.Println(err.Error())
		c.SendStatus(fiber.StatusInternalServerError)
	}
	c.SendString(string(json))
}

// < ----- GET ROUTES ----- >

// Login is the frontend used to both signin and signup.
func (server *Server) Login(c *fiber.Ctx) {
	if c.Path() == "/signup" {
		if err := c.Render("./views/login.pug", fiber.Map{"signup": true}); err != nil {
			c.Status(500).Send(err.Error())
		}
	} else {
		if err := c.Render("./views/login.pug", fiber.Map{"signup": false}); err != nil {
			c.Status(500).Send(err.Error())
		}
	}
}

// Home shows all the current files in the given directory.
func (server *Server) Home(c *fiber.Ctx) {
	// Get current user information from the claims map.
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)

	// Setup the query path.
	qPath := server.Volume.Path
	if len(c.Query("path")) == 0 {
		qPath += "/"
	} else {
		qPath += c.Query("path")
	}
	// Walk the given folder and return a list of files
	files, err := server.Volume.WalkFolder(qPath)
	if err != nil {
		fmt.Println(err.Error())
	}
	// Generate the path object used in the breadcrumbs.
	paths := strings.Split(qPath, "./")
	paths = strings.Split(paths[1], "/")
	paths = deleteEmpty(paths)
	if err != nil {
		c.Status(500).Send(err.Error())
	}
	endpath := ""
	Volumepath := ""
	if len(paths) < 2 {
		endpath = server.Volume.Name
		paths = nil
	} else {
		Volumepath = server.Volume.Name
		endpath = paths[len(paths)-1]
		paths = paths[1 : len(paths)-1]
	}
	// Generate the bindings for the amber template
	tUser := server.GetUserByUsername(claims["username"].(string))

	settingsMap := tUser.FileSettings.ToMap()
	files = files.AddFileSetting(settingsMap, server.IconsList)
	bind := fiber.Map{
		"user":       tUser,
		"files":      files,
		"volumepath": Volumepath,
		"paths":      paths,
		"endpath":    endpath,
	}
	// Render the amber template. It's .pug because i needed syntax higlighting
	if err = c.Render("./views/home.pug", bind); err != nil {
		c.Status(500).Send(err.Error())
	}
}

// Settings is the page where you configure the icons and the webapps used for the diffrent file types. Atleast when it's implementet
func (server *Server) Settings(c *fiber.Ctx) {
	// Get current user information from the claims map.
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	tUser := server.GetUserByUsername(claims["username"].(string))
	bind := fiber.Map{
		"user":         tUser,
		"fileSettings": tUser.FileSettings,
	}
	if err := c.Render("./views/settings.pug", bind); err != nil {
		c.Status(500).Send(err.Error())
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

// JwtErrorHandler handles errors involving JWT
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
	server.DB, err = sql.Open("sqlite3", "file:./db/database.db?_auth&_auth_user="+server.Username+"&_auth_pass="+server.Password+"&_auth_crypt=sha256&cache=shared")
	if err != nil {
		panic(err)
	}
	// Protects the database by locking it to a single connection
	server.DB.SetMaxOpenConns(1)

	// Setup the user table if it doesn't exist'
	statement, err := server.DB.Prepare(`
		CREATE TABLE IF NOT EXISTS Users(
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

	// Setup the FileSettings table if it doesn't exist'
	statement, err = server.DB.Prepare(`
		CREATE TABLE IF NOT EXISTS FileSettings(
			ID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			Username TEXT,
			Extension TEXT,
			ApplicationLink TEXT,
			Icon TEXT
		);
	`)
	if err != nil {
		panic(err)
	}
	statement.Exec()

	// Setup the pdfs table if it doesn't exist'
	statement, err = server.DB.Prepare(`
		CREATE TABLE IF NOT EXISTS PDFS(
			ID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			Username TEXT,
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

// < ----- USER DB START ----- >

// InsertUser inserts a user into the database.
func (server *Server) InsertUser(username string, password string, profilepicture string) error {
	statement, _ := server.DB.Prepare("INSERT INTO Users (Username, Password, ProfilePicture) values (?,?,?)")
	_, err := statement.Exec(username, password, profilepicture)
	if err != nil {
		return err
	}
	return nil
}

// GetUserByUsername gets the user by their username and returns the user as a User object.
func (server *Server) GetUserByUsername(username string) User {
	result := server.DB.QueryRow("select * from Users where Username=$1", username)
	user := User{}
	result.Scan(&user.ID, &user.Username, &user.Password, &user.ProfilePicture)
	user.FileSettings = server.GetFileSettingsByUsername(username)
	return user
}

// < ----- FILESETTINGS DB START ----- >

// InsertFileSetting inserts a filesetting into the database.
func (server *Server) InsertFileSetting(Username string, Extension string, ApplicationLink string) error {
	statement, _ := server.DB.Prepare("INSERT INTO FileSettings (Username, Extension, Icon, ApplicationLink) values (?,?,?,?)")
	_, err := statement.Exec(Username, Extension, "", ApplicationLink)
	if err != nil {
		return err
	}
	return nil
}

// UpdateFileSetting updates a FileSetting if the username and extension exits in the database. If it's not in the database the FileSetting will be insertet into the database.
func (server *Server) UpdateFileSetting(Username string, Extension string, ApplicationLink string) error {
	statement, _ := server.DB.Prepare(`
		UPDATE FileSettings SET ApplicationLink=$1 WHERE Extension=$2 AND Username=$3
	`)
	result, err := statement.Exec(ApplicationLink, Extension, Username)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		err := server.InsertFileSetting(Username, Extension, ApplicationLink)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetFileSettingByUsernameAndExtension returns a single file settings for a given user and extension.
func (server *Server) GetFileSettingByUsernameAndExtension(Username string, Extension string) Files.FileSetting {
	result := server.DB.QueryRow("SELECT * FROM FileSettings WHERE Username=$1 AND Extension=$2", Username, Extension)
	setting := Files.FileSetting{}
	result.Scan(&setting.ID, &setting.Username, &setting.Extension, &setting.ApplicationLink, &setting.Icon)
	return setting
}

// GetFileSettingsByUsername returns a list of all file settings for a given user.
func (server *Server) GetFileSettingsByUsername(Username string) Files.FileSettings {
	result, err := server.DB.Query("SELECT * FROM FileSettings WHERE Username=$1", Username)
	if err != nil {
		panic(err)
	}
	fileSettings := Files.FileSettings{}
	for result.Next() {
		setting := Files.FileSetting{}
		err := result.Scan(&setting.ID, &setting.Username, &setting.Extension, &setting.ApplicationLink, &setting.Icon)
		if err != nil {
			fmt.Println(err.Error())
		}
		fileSettings = append(fileSettings, setting)
	}
	return fileSettings
}

// < ----- Helpers ----- >
func deleteEmpty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}
