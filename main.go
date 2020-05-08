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
		"Routes":{
				"updatePageCount": {
					"Path":"/updatePageCount",
					"Methode":"POST",
					"View":"",
					"FormValueNames": [],
					"ParameterNames": [],
					"DatabaseQueries": []
				},
				"getPDF":{
					"Path":"/GetPdfByHash",
					"Methode":"POST",
					"View":"",
					"FormValueNames": [],
					"ParameterNames": ["HASH"],
					"DatabaseQueries": []
				},
		},
		"DatabaseQueries":{
			"UpdatePageCount":{
				"Result":{

				},
				"Contains":{
					"Hash":"Hash",
					"Username":"Hash"
				},
				"Set":{
					"Page":"PageNumber"
				},
				"TableName":"PDFS",
				"DatabaseOperation":"UPDATE"
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
	dumpMap("", jsonMap)
}

func dumpMap(space string, m map[string]interface{}) {
	var Extension Server.Extension
	for k, v := range m {
		if k == "Name" {
			Extension.Name = v.(string)
		}
		if k == "Routes" {
			//v.(map[string]interface{}
			var Routes []Server.Route
			if mv, ok := v.(map[string]interface{}); ok {
				for _, v := range mv {
					if mv2, ok := v.(map[string]interface{}); ok {
						Routes = append(Routes, dumpRoute(mv2))
					}
				}
				Extension.Routes = Routes
			}
		}
		if k == "DatabaseQueries" {
			//v.(map[string]interface{}
			var DatabaseQueries []Server.DatabaseQuery
			if mv, ok := v.(map[string]interface{}); ok {
				for _, v := range mv {
					if mv2, ok := v.(map[string]interface{}); ok {
						DatabaseQueries = append(DatabaseQueries, dumpDatabaseQuery(mv2))
					}
				}
				//Extension = DatabaseQueries
			}
		}
	}
}

func dumpRoute(m map[string]interface{}) Server.Route {
	var Route Server.Route
	for k, v := range m {
		fmt.Println("key: ", k, "value: ", v)
		switch k {
		case "Path":
			Route.Path = v.(string)
		case "View":
			Route.View = v.(string)
		case "DatabaseQueries":
			Route.DatabaseQueries = v.([]Server.DatabaseQuery)
		}
	}
	return Route
}

func dumpDatabaseQuery(m map[string]interface{}) Server.DatabaseQuery {
	return Server.DatabaseQuery{}
}
