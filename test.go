package test

import (
	"flag"
	"log"

	"./api"
	"./backend"
)

// Flags
var (
	port             = flag.String("port", "80", "Sets the port of the API server.")
	scheme           = flag.String("scheme", "http", "Sets the scheme of the API server.")
	driverName       = flag.String("driverName", "sqlite3", "Sets the driverName of the Database.")
	connectionString = flag.String("connectionString", "./db.sql", "Sets the connectionString used to connect to the database.")
	certificate      = flag.String("cert", "./key/cert", "Sets the path to the ssl certificate.")
	keyfile          = flag.String("key", "./key/key", "Sets the path to the ssl key file.")
)

func main() {
	flag.Parse()
	//go initBackend()
	initAPI()
}

func initAPI() {
	api := &api.API{}
	err := api.Init(*port, *scheme)
	if err != nil {
		log.Fatal(err)
	}
	println(*certificate)
	println(*keyfile)
}
func initBackend() {
	backend := &backend.Backend{}
	backend.InitDB(*driverName, *connectionString)
}
