package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	ExtensionAPI "./libs/extension"
	files "./libs/files"
	Icons "./libs/icons"
	Server "./libs/server"

	"github.com/gofiber/fiber"
	jwtware "github.com/gofiber/jwt"
	"github.com/gofiber/logger"
	"github.com/gofiber/template"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Setup the server
	server := &Server.Server{}
	flags(server)
	// Setup the database
	server.InitDB()
	server.IconsList = Icons.GenerateIconsList()
	// setup the volume for the server.
	server.Volume = files.Volume{Name: "C:", Path: "./files"}
	// Setup fiber
	app := fiber.New()
	app.Settings.TemplateEngine = template.Amber()
	// setup logger middleware
	app.Use(logger.New())

	// < ----- STATIC ROUTES ----- >

	app.Static("/css", "./styles/css")
	app.Static("/js", "./js")
	app.Static("/media", "./media")

	// < ----- GET ROUTES ----- >

	app.Get("/", server.Login)
	app.Get("/signin", server.Login)
	app.Get("/signup", server.Login)

	// < ----- POST ROUTES ----- >

	app.Post("/signin", server.Signin)
	app.Post("/signup", server.Signup)

	// < ----- PROTECTET ROUTES ----- >

	app.Use(jwtware.New(jwtware.Config{
		SigningKey:   []byte(server.Secret),
		TokenLookup:  "cookie:token",
		ErrorHandler: server.JwtErrorHandler,
	}))

	// < ----- STATIC ROUTES ----- >

	app.Static("/volume", server.Volume.Path)

	// < ----- GET ROUTES ----- >

	app.Get("/home", server.Home)
	app.Get("/settings", server.Settings)
	app.Get("/files", server.GetFiles)

	// < ----- POST ROUTES ----- >

	app.Post("/updateSetting", server.UpdateSetting)
	app.Post("/query", server.Query)

	// < ----- EXTENSIONS ----- >

	Extensions := ExtensionAPI.Extensions{}
	Extensions.LoadExtensions(app, server.DB)

	// < ----- TEST ----- >
	test(server.DB, app)
	// start the server on the server.port
	log.Fatal(app.Listen(server.Port))
}

// < ----- FLAGS ----- >

func flags(server *Server.Server) {
	secret := flag.String("secret", stringWithCharset(256, charset), "The secrect is a key used for signing the users JWT token")
	port := flag.Int("port", 8080, "The port is the port used to server the server")
	username := flag.String("username", "admin", "The Username is for the database to ensure the data is protected")
	password := flag.String("password", "admin", "The Password is for the database to ensure the data is protected")
	flag.Parse()
	server.Secret = *secret
	server.Port = *port
	server.Username = *username
	server.Password = *password
}

// < ----- Random Generators ----- >
const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const charset2 = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func stringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func test(DB *sql.DB, app *fiber.App) {
	//databaseTest(DB)
	//viewTest(DB, app)
	//configTest()
}

func databaseTest(DB *sql.DB) {
	dataBaseItems := ExtensionAPI.DatabaseItems{}
	for i := 0; i < 100; i++ {
		dataBaseItems[stringWithCharset(5, charset2)] = ExtensionAPI.TEXT
	}
	// Generate the database table.
	dataBaseTable := ExtensionAPI.DatabaseTable{TableName: "TestTable", Items: dataBaseItems}
	err := dataBaseTable.GenerateTable(DB)
	if err != nil {
		fmt.Println(err.Error())
	}
	// Add item to a table
	DatabaseQuery := ExtensionAPI.DatabaseQuery{TableName: "PDFS", DatabaseOperation: ExtensionAPI.INSERT, Contains: make(map[string]string)}
	DatabaseQuery.Contains["Username"] = "\"LowkeyCoding\""
	DatabaseQuery.Contains["Hash"] = "\"somehash\""
	DatabaseQuery.Contains["Path"] = "\"/test\""
	DatabaseQuery.Contains["Page"] = "1"

	_, err = DatabaseQuery.GenerateQuery(DB)
	if err != nil {
		fmt.Println(err.Error())
	}

	// Get the item from a table
	DatabaseQuery = ExtensionAPI.DatabaseQuery{TableName: "PDFS", DatabaseOperation: ExtensionAPI.SELECT, Contains: make(map[string]string)}
	DatabaseQuery.Contains["Hash"] = "\"somehash\""

	_, err = DatabaseQuery.GenerateQuery(DB)
	if err != nil {
		fmt.Println(err.Error())
	}

	// Update item form a table
	DatabaseQuery = ExtensionAPI.DatabaseQuery{TableName: "PDFS", DatabaseOperation: ExtensionAPI.UPDATE, Contains: make(map[string]string), Set: make(map[string]string)}
	DatabaseQuery.Contains["Hash"] = "\"somehash\""
	DatabaseQuery.Set["Page"] = "2"

	_, err = DatabaseQuery.GenerateQuery(DB)
	if err != nil {
		fmt.Println(err.Error())
	}

	// Get the item from a table
	DatabaseQuery = ExtensionAPI.DatabaseQuery{TableName: "PDFS", DatabaseOperation: ExtensionAPI.SELECT, Contains: make(map[string]string)}
	DatabaseQuery.Contains["Hash"] = "\"somehash\""

	_, err = DatabaseQuery.GenerateQuery(DB)
	if err != nil {
		fmt.Println(err.Error())
	}

	// Delete item form a table
	DatabaseQuery = ExtensionAPI.DatabaseQuery{TableName: "PDFS", DatabaseOperation: ExtensionAPI.DELETE, Contains: make(map[string]string)}
	DatabaseQuery.Contains["Hash"] = "\"somehash\""

	_, err = DatabaseQuery.GenerateQuery(DB)
	if err != nil {
		fmt.Println(err.Error())
	}

	// Get the item from a table
	DatabaseQuery = ExtensionAPI.DatabaseQuery{TableName: "PDFS", DatabaseOperation: ExtensionAPI.SELECT, Contains: make(map[string]string)}
	DatabaseQuery.Contains["Hash"] = "\"somehash\""

	_, err = DatabaseQuery.GenerateQuery(DB)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func viewTest(DB *sql.DB, app *fiber.App) {
	// Create the viewQueryVariableNames array
	var viewQueryVariableNames []string
	viewQueryVariableNames = append(viewQueryVariableNames, "Hash")
	viewQueryVariableNames = append(viewQueryVariableNames, "Username")
	// Set query Variables
	var queryContains map[string]string
	queryContains = make(map[string]string)
	queryContains["Hash"] = ""
	queryContains["Username"] = ""
	// Set variable types
	var VariableType map[string]ExtensionAPI.DatabaseItemType
	VariableType = make(map[string]ExtensionAPI.DatabaseItemType)
	VariableType["Hash"] = ExtensionAPI.TEXT
	VariableType["Username"] = ExtensionAPI.TEXT
	pdfReaderView := ExtensionAPI.View{
		Path:               "/pdf",
		ViewPath:           "./views/pdf.pug",
		NeedsQuerying:      true,
		QueryVariableNames: viewQueryVariableNames,
		DatabaseQuery: ExtensionAPI.DatabaseQuery{
			TableName:         "PDFS",
			VariableType:      VariableType,
			Contains:          queryContains,
			DatabaseOperation: ExtensionAPI.SELECT,
		},
	}
	var items ExtensionAPI.DatabaseItems
	items = make(map[string]ExtensionAPI.DatabaseItemType)

	items["ID"] = ExtensionAPI.INTEGER
	items["Username"] = ExtensionAPI.TEXT
	items["Hash"] = ExtensionAPI.TEXT
	items["Path"] = ExtensionAPI.TEXT
	items["Page"] = ExtensionAPI.INTEGER
	databaseTable := ExtensionAPI.DatabaseTable{
		TableName: "PDFS",
		Items:     items,
	}
	// Add the view to a list of views
	var pdfReaderViews []ExtensionAPI.View
	pdfReaderViews = append(pdfReaderViews, pdfReaderView)
	// Add the database table to a list of database tables
	var pdfDatabaseTables []ExtensionAPI.DatabaseTable
	pdfDatabaseTables = append(pdfDatabaseTables, databaseTable)
	extension := ExtensionAPI.Extension{
		Name:           "PDFREADER",
		Views:          pdfReaderViews,
		DatabaseTables: pdfDatabaseTables,
	}
	for _, databaseTable := range extension.DatabaseTables {
		databaseTable.GenerateTable(DB)
	}
	for _, view := range extension.Views {
		view.GenerateView(app, DB)
	}
	test, err := json.Marshal(&extension)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(string(test))
}

func configTest() {
	config := `
	{
		"Name": "PDFREADER",
		"Views":{
			"View": {
				"Path": "/pdf",
				"ViewPath": "./views/pdf.pug",
				"NeedsQuerying": true,
				"QueryVariableNames": [
				  "Hash",
				  "Username"
				],
				"DatabaseQuery": {
				  "Result": null,
				  "VariableType": {
					"Hash": "TEXT",
					"Username": "TEXT"
				  },
				  "Contains": {
					"Hash": "",
					"Username": ""
				  },
				  "Set": {},
				  "TableName": "PDFS",
				  "DatabaseOperation": "SELECT"
				}
			}
		},
		"DatabaseTables": {
			"DatabaseTable":{
				"TableName": "PDFS",
				"Items": {
				  "Hash": "TEXT",
				  "ID": "INTEGER",
				  "Page": "INTEGER",
				  "Path": "TEXT",
				  "Username": "TEXT"
				}
			}
		}
	}
	`
	jsonMap := make(map[string]interface{})
	err := json.Unmarshal([]byte(config), &jsonMap)
	if err != nil {
		panic(err)
	}
	var Extension ExtensionAPI.Extension
	Extension = Extension.InterfaceToExtension("", jsonMap)
	fmt.Println(Extension)
}
