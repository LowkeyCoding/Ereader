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
	app.Get("/pdf-viewer", server.PdfViewer)

	// < ----- POST ROUTES ----- >

	app.Post("/updateSetting", server.UpdateSetting)
	app.Post("/query", server.Query)
	app.Post("/pdf-update", server.PdfUpdate)

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
	viewTest(DB, app)
	//configTest() // Fails since it's a hughe struggle
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
	test, err := json.Marshal(&pdfReaderView)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(string(test))
}

func configTest() {
	config := `
	{
		"Name":"PDFReader",
		"Views":{
				"updatePageCount": {
					"Path":"/updatePageCount",
					"ViewPath":"",
					"NeedsQuerying":false,
					"QueryVariableNames":  ["Hash", "Username"],
					"DatabaseQuery": {
							"Result":{
								
							},
							"Contains":{
								"Hash":"Hash",
								"Username":"Username"
							},
							"Set":{
								"Page":"PageNumber"
							},
							"TableName":"PDFS",
							"DatabaseOperation":"UPDATE"
					}
				},
				"getPDF":{
					"Path":"/GetPdfByHash",
					"ViewPath":"",
					"NeedsQuerying":true,
					"QueryVariableNames": ["Hash", "Username"],
					"DatabaseQuery": {
						"Result":{
							
						},
						"Contains":
							"Hash":"Hash",
							"Username":"Username"
						},
						"Set":{},
						"TableName":"PDFS",
						"DatabaseOperation":"SELECT"
					}
				}
		},
		"DatabaseTables":{
			"PDFS":{
				"ID":"INTERGER",
				"Username":"TEXT",
				"Hash":"TEXT",
				"Path":"TEXT",
				"Page":"INTEGER"
			}
		}
	 }
	`
	jsonMap := make(map[string]interface{})
	err := json.Unmarshal([]byte(config), &jsonMap)
	if err != nil {
		panic(err)
	}
	Extension := InterfaceToExtension("", jsonMap)
	fmt.Println("Extension: ", Extension)
}

// InterfaceToExtension converts an interface to the Extension structure.
func InterfaceToExtension(space string, m map[string]interface{}) ExtensionAPI.Extension {
	var Extension ExtensionAPI.Extension
	for k, v := range m {
		if k == "Name" {
			Extension.Name = v.(string)
		} else if k == "Views" {
			//v.(map[string]interface{}
			var Views []ExtensionAPI.View
			if mv, ok := v.(map[string]interface{}); ok {
				for _, v := range mv {
					if mv2, ok := v.(map[string]interface{}); ok {
						Views = append(Views, InterfaceToView(mv2))
					}
				}
				Extension.Views = Views
			}
		} else if k == "DatabaseTables" {
			//v.(map[string]interface{}
			var DatabaseQueries []ExtensionAPI.DatabaseTable
			if mv, ok := v.(map[string]interface{}); ok {
				for _, v := range mv {
					if mv2, ok := v.(map[string]interface{}); ok {
						DatabaseQueries = append(DatabaseQueries, InterfaceToTable(mv2))
					}
				}
				Extension.DatabaseTables = DatabaseQueries
			}
		}
	}
	return Extension
}

// InterfaceToView converts an interface to the View structure.
func InterfaceToView(m map[string]interface{}) ExtensionAPI.View {
	var View ExtensionAPI.View
	for k, v := range m {
		switch k {
		case "Path":
			View.Path = v.(string)
		case "ViewPath":
			View.ViewPath = v.(string)
		case "NeedsQuerying":
			View.NeedsQuerying = v.(bool)
		case "QueryVariableNames":
			/*fmt.Println("QueryVariableNames", v.(map[string]interface{}))
			for kk, vv := range v.(map[string]interface{}) {
				fmt.Println("Key", kk, "Value", vv)
			}*/
			xType := fmt.Sprintf("%T", v)
			fmt.Println(xType)
			//View.QueryVariableNames = v.([]string)
		case "DatabaseQueries":
			if mv, ok := v.(map[string]interface{}); ok {
				View.DatabaseQuery = InterfaceToQuery(mv)
			}
		}
	}
	fmt.Println("View: ", View)
	return View
}

// InterfaceToQuery converts an interface to the DatabaseQuery structure.
func InterfaceToQuery(m map[string]interface{}) ExtensionAPI.DatabaseQuery {
	var Query ExtensionAPI.DatabaseQuery
	for k, v := range m {
		switch k {
		case "Contains":
			Query.Contains = v.(map[string]string)
		case "Set":
			Query.Set = v.(map[string]string)
		case "TableName":
			Query.TableName = v.(string)
		case "DatabaseOperation":
			Query.DatabaseOperation = v.(ExtensionAPI.DatabaseOperationType)
		}
	}
	fmt.Println("Query: ", Query)
	return Query
}

// InterfaceToTable converts an interface to the DatabaseTable structure.
func InterfaceToTable(m map[string]interface{}) ExtensionAPI.DatabaseTable {
	var DatabaseTable ExtensionAPI.DatabaseTable
	for k, v := range m {
		switch k {
		case "TableName":
			DatabaseTable.TableName = v.(string)
		case "Items":
			DatabaseTable.Items = v.(ExtensionAPI.DatabaseItems)
		}
	}
	fmt.Println("DatabaseTable: ", DatabaseTable)
	return DatabaseTable
}
