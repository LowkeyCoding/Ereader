package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

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
	test(server.DB)
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

func test(DB *sql.DB) {
	//databaseTest(DB)
	configTest() // Fails since it's a hughe struggle
}

func databaseTest(DB *sql.DB) {
	dataBaseItems := Server.DatabaseItems{}
	for i := 0; i < 100; i++ {
		dataBaseItems[stringWithCharset(5, charset2)] = Server.TEXT
	}
	dataBaseTable := Server.DatabaseTable{TableName: "TestTable", Items: dataBaseItems}
	err := dataBaseTable.GenerateTable(DB)
	if err != nil {
		fmt.Println(err.Error())
	}

	DatabaseQuery := Server.DatabaseQuery{TableName: "PDFS", DatabaseOperation: Server.INSERT, Contains: make(map[string]string)}
	DatabaseQuery.Contains["Username"] = "\"LowkeyCoding\""
	DatabaseQuery.Contains["Hash"] = "\"somehash\""
	DatabaseQuery.Contains["Path"] = "\"/test\""
	DatabaseQuery.Contains["Page"] = "1"

	_, err = DatabaseQuery.GenerateQuery(DB)
	if err != nil {
		fmt.Println(err.Error())
	}
	DatabaseQuery = Server.DatabaseQuery{TableName: "PDFS", DatabaseOperation: Server.UPDATE, Contains: make(map[string]string), Set: make(map[string]string)}
	DatabaseQuery.Contains["Hash"] = "\"somehash\""
	DatabaseQuery.Set["Page"] = "2"

	_, err = DatabaseQuery.GenerateQuery(DB)
	if err != nil {
		fmt.Println(err.Error())
	}

	DatabaseQuery = Server.DatabaseQuery{TableName: "PDFS", DatabaseOperation: Server.SELECT, Contains: make(map[string]string)}
	DatabaseQuery.Contains["Hash"] = "\"somehash\""

	_, err = DatabaseQuery.GenerateQuery(DB)
	if err != nil {
		fmt.Println(err.Error())
	}

	DatabaseQuery = Server.DatabaseQuery{TableName: "PDFS", DatabaseOperation: Server.DELETE, Contains: make(map[string]string)}
	DatabaseQuery.Contains["Hash"] = "\"somehash\""

	_, err = DatabaseQuery.GenerateQuery(DB)
	if err != nil {
		fmt.Println(err.Error())
	}

	DatabaseQuery = Server.DatabaseQuery{TableName: "PDFS", DatabaseOperation: Server.SELECT, Contains: make(map[string]string)}
	DatabaseQuery.Contains["Hash"] = "\"somehash\""

	_, err = DatabaseQuery.GenerateQuery(DB)
	if err != nil {
		fmt.Println(err.Error())
	}
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
	fmt.Println(Extension)
}

// InterfaceToExtension converts an interface to the Extension structure.
func InterfaceToExtension(space string, m map[string]interface{}) Server.Extension {
	var Extension Server.Extension
	for k, v := range m {
		if k == "Name" {
			Extension.Name = v.(string)
		} else if k == "Views" {
			//v.(map[string]interface{}
			var Views []Server.View
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
			var DatabaseQueries []Server.DatabaseTable
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
func InterfaceToView(m map[string]interface{}) Server.View {
	var View Server.View
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
	return View
}

// InterfaceToQuery converts an interface to the DatabaseQuery structure.
func InterfaceToQuery(m map[string]interface{}) Server.DatabaseQuery {
	var Query Server.DatabaseQuery
	for k, v := range m {
		switch k {
		case "Contains":
			Query.Contains = v.(map[string]string)
		case "Set":
			Query.Set = v.(map[string]string)
		case "TableName":
			Query.TableName = v.(string)
		case "DatabaseOperation":
			Query.DatabaseOperation = v.(Server.DatabaseOperationType)
		}
	}
	return Query
}

// InterfaceToTable converts an interface to the DatabaseTable structure.
func InterfaceToTable(m map[string]interface{}) Server.DatabaseTable {
	var DatabaseTable Server.DatabaseTable
	for k, v := range m {
		switch k {
		case "TableName":
			DatabaseTable.TableName = v.(string)
		case "Items":
			DatabaseTable.Items = v.(Server.DatabaseItems)
		}
	}
	return DatabaseTable
}
