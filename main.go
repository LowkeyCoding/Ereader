package main

import (
	"flag"
	"log"
	"math/rand"
	"time"

	files "./libs/files"
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
	// setup the volume for the server.
	server.Volume = files.Volume{Name: "C:", Path: "./files"}
	// Setup fiber
	app := fiber.New()
	app.Settings.TemplateEngine = template.Amber()
	// setup logger middleware
	app.Use(logger.New())
	// setup static routes
	app.Static("/css", "./styles/css")
	app.Static("/js", "./js")
	app.Static("/media", "./media")
	app.Static("/volume", server.Volume.Path)
	// setup GET routes
	app.Get("/signin", server.Login)
	app.Get("/signup", server.Login)
	// setup POST routes
	app.Post("/signin", server.Signin)
	app.Post("/signup", server.Signup)
	// setup authentication routes
	app.Use(jwtware.New(jwtware.Config{
		SigningKey:   []byte(server.Secret),
		TokenLookup:  "cookie:token",
		ErrorHandler: server.JwtErrorHandler,
	}))
	// setup GET routes
	app.Get("/home", server.Home)
	app.Get("/pdf-viewer", server.PdfViewer)
	app.Post("/pdf-update", server.PdfUpdate)
	// start the server on the server.port
	log.Fatal(app.Listen(server.Port))
}

// < ----- FLAGS ----- >

func flags(server *Server.Server) {
	secret := flag.String("secret", stringWithCharset(256, charset), "The secrect is a key used for signing the users JWT token")
	port := flag.Int("port", 8080, "The port is the port used to server the server")
	flag.Parse()
	server.Secret = *secret
	server.Port = *port
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
