package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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

// < ----- PDF ----- >

// PDF is a struct representing a pdf in the database
type PDF struct {
	ID       string
	Username string
	Hash     string
	Path     string
	Page     int
}

// < ----- Server ----- >

// Server class
type Server struct {
	DB        *sql.DB
	Username  string
	Password  string
	Secret    string
	Port      int
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

// < ----- PDF EXTENSION ROUTE START ----- >

// PdfViewer servers the page that renders the pdf.
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
		pdf.Username = claims["username"].(string)
		err := server.InsertPdf(pdf.Username, pdf.Hash, pdf.Path, pdf.Page)
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
	if err := c.Render("./views/pdf-viewer.pug", bind); err != nil {
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

// < ----- PDF EXTENSION ROUTE STOP ----- >

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
			Icon TEXT,
			ApplicationLink TEXT
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
	result.Scan(&setting.ID, &setting.Username, &setting.Extension, &setting.Icon, &setting.ApplicationLink)
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
		err := result.Scan(&setting.ID, &setting.Username, &setting.Extension, &setting.Icon, &setting.ApplicationLink)
		if err != nil {
			fmt.Println(err.Error())
		}
		fileSettings = append(fileSettings, setting)
	}
	return fileSettings
}

// < ----- PDF EXTENSION DB START ----- >

// InsertPdf inserts a pdf into the database
func (server *Server) InsertPdf(Username string, hash string, path string, page int) error {
	statement, _ := server.DB.Prepare(`
		INSERT INTO PDFS (Username, Hash, Path, Page) values (?,?,?,?)
	`)
	_, err := statement.Exec(Username, hash, path, page)
	if err != nil {
		return err
	}
	return nil
}

// UpdatePdfPageCount updates the current page count of a given pdf.
func (server *Server) UpdatePdfPageCount(Username string, hash string, path string, page int) error {
	statement, _ := server.DB.Prepare(`
		UPDATE PDFS SET Page=$1 WHERE Hash=$2 AND Username=$3
	`)
	result, err := statement.Exec(page, hash, Username)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		err := server.InsertPdf(Username, hash, path, page)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetPdfByHash gets the pdf from it's hash and returns it as a PDF object.
func (server *Server) GetPdfByHash(Username string, hash string) PDF {
	result := server.DB.QueryRow("SELECT * FROM PDFS WHERE Username=$1 AND Hash=$2", Username, hash)
	pdf := PDF{}
	result.Scan(&pdf.ID, &pdf.Username, &pdf.Hash, &pdf.Path, &pdf.Page)
	return pdf
}

// < ----- PDF EXTENSION DB STOP ----- >

// < ----- Extension Generators ----- >

//Extension ..
type Extension struct {
	Routes          []Route
	DatabaseQueries []DatabaseQuery
	DatabaseTables  []DatabaseTable
	Structs         map[string][]map[string]interface{}
}

// Route ..
type Route struct {
	Path            string          // The path the view will be rendered to.
	View            string          // The view name. Will be used to select the correct view to render.
	FormValueNames  []string        // The result will be passed to the tempalte generator or the database queries depending on the config.
	ParameterNames  []string        // The result will be passed to the tempalte generator or the database queries depending on the config.
	DatabaseQueries []DatabaseQuery // The result will be passed to the tempalte generator.
	config          string          // well dunno how the config is gonna work yet...
}

// DatabaseItemType Defines the allowed types of values for the database.
type DatabaseItemType string

func (itemType *DatabaseItemType) String() string {
	switch *itemType {
	case NULL:
		return "NULL"
	case INTEGER:
		return "INTEGER"
	case REAL:
		return "REAL"
	case TEXT:
		return "TEXT"
	case BLOB:
		return "BLOB"
	}
	return ""
}

const (
	// NULL Sqlite3. The value is a NULL value
	NULL DatabaseItemType = "NULL"
	// INTEGER Sqlite3. The value is a signed integer, stored in 1, 2, 3, 4, 6, or 8 bytes depending on the magnitude of the value.
	INTEGER DatabaseItemType = "INTEGER"
	// REAL Sqlite3. The value is a floating point value, stored as an 8-byte IEEE floating point number.
	REAL DatabaseItemType = "REAL"
	// TEXT Sqlite3. The value is a text string, stored using the database encoding (UTF-8, UTF-16BE or UTF-16LE).
	TEXT DatabaseItemType = "TEXT"
	// BLOB Sqlite3. The value is a blob of data, stored exactly as it was input.
	BLOB DatabaseItemType = "BLOB"
)

// DatabaseOperationType Defines the operation types for the database.
type DatabaseOperationType string

func (operationType *DatabaseOperationType) String() string {
	switch *operationType {
	case INSERT:
		return "INSERT"
	case SELECT:
		return "SELECT"
	case UPDATE:
		return "UPDATE"
	case DELETE:
		return "DELETE"
	}
	return ""
}

const (
	// INSERT sqlite3. SQLite INSERT INTO Statement is used to add new rows of data into a table in the database.
	INSERT DatabaseOperationType = "INSERT"
	// SELECT sqlite3. SQLite SELECT statement is used to fetch the data from a SQLite database table which returns data in the form of a result table.
	SELECT DatabaseOperationType = "SELECT"
	// UPDATE sqlite3. SQLite UPDATE statement is used to  modify the existing records in a table. You can use WHERE clause with UPDATE query to update selected rows, otherwise all the rows would be updated.
	UPDATE DatabaseOperationType = "UPDATE"
	// DELETE statement is used to delete  the existing records from a table. You can use WHERE clause with DELETE query to delete the selected rows, otherwise all the records would be deleted.
	DELETE DatabaseOperationType = "DELETE"
)

// DatabaseItems is a the item insertet into the database table.
// This is made so that a the database item name can be mapped to it's type.
type DatabaseItems map[string]DatabaseItemType

//DatabaseTable ..
type DatabaseTable struct {
	TableName string
	Items     DatabaseItems
}

// GenerateTable generates the given database table if it doesn't exist'.
func (database *DatabaseTable) GenerateTable(DB *sql.DB) error {
	// Setup the FileSettings table if it doesn't exist'
	Query := "CREATE TABLE IF NOT EXISTS " + database.TableName + "("
	Query += "ID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT"
	i := 0
	for key, sqlType := range database.Items {
		if i < len(database.Items) {
			Query += ", " + key + " " + sqlType.String()
		} else {
			Query += ", " + key + " " + sqlType.String()
		}
		i++
	}
	Query += ");"
	statement, err := DB.Prepare(Query)
	if err != nil {
		return err
	}
	statement.Exec()
	return nil
}

//DatabaseQuery ..
type DatabaseQuery struct {
	Result            []map[string]interface{}
	Contains          map[string]string
	TableName         string
	DatabaseOperation DatabaseOperationType
}

// Query constructs the query based on the information provided from the DatabaseQuery
func (query *DatabaseQuery) Query(DB *sql.DB) error {
	Query := ""
	i := 0
	switch query.DatabaseOperation {
	case INSERT:
		//INSERT INTO PDFS (Username, Hash, Path, Page) values (?,?,?,?)
		Query = query.DatabaseOperation.String() + " INTO " + query.TableName + " ("
		keyMap := make(map[int]string)
		i = 0
		for key := range query.Contains {
			keyMap[i] = key
			if i < len(query.Contains)-1 {
				Query += key + ","
			} else {
				Query += key + ")"
			}
			i++
		}
		Query += " VALUES ("
		for j := 0; j < i; j++ {
			if j < i-1 {
				Query += query.Contains[keyMap[j]] + ","
			} else {
				Query += query.Contains[keyMap[j]] + ")"
			}
		}

	case SELECT:
		// SELECT * FROM PDFS WHERE Username=$1 AND Hash=$2
		Query = query.DatabaseOperation.String() + " * FROM " + query.TableName
		i = 0
		if len(query.Contains) > 0 {
			Query += " WHERE "
			for key, value := range query.Contains {
				if i < len(query.Contains) {
					Query += key + "=" + value
				} else {
					Query += key + "=" + value + " AND "
				}
				i++
			}
		}
	}
	fmt.Println("Query: ", Query)
	rows, err := DB.Query(Query)
	if err != nil {
		return err
	}
	query.LoadResultIntoMap(rows)
	return nil
}

// LoadResultIntoMap ..
func (query *DatabaseQuery) LoadResultIntoMap(rows *sql.Rows) error {
	var columns []string
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	colNum := len(columns)

	for rows.Next() {
		// Prepare to read row using Scan
		r := make([]interface{}, colNum)
		for i := range r {
			r[i] = &r[i]
		}

		// Read rows using Scan
		err = rows.Scan(r...)
		if err != nil {
			return err
		}

		// Create a row map to store row's data
		var row = map[string]interface{}{}
		for i := range r {
			row[columns[i]] = r[i]
		}

		// Append to the final results slice
		query.Result = append(query.Result, row)
	}

	fmt.Println(query.Result) // You can then json.Marshal or w/e

	// If you want it pretty-printed
	r, err := json.MarshalIndent(query.Result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(r))
	return nil
}

func (server *Server) generateRoutes(app *fiber.App) {
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
