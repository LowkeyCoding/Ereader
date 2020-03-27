package backend

import (
	"database/sql"
	"hash"
	"log"
)

// Backend implements the database interface for the user frontend.
type Backend struct {
	db *sql.DB
}

type dbFile struct {
	path     string
	name     string
	mimeType string
	siSize   string
	size     int64

	hash hash.Hash

	isFolder   bool
	isReadOnly bool

	acess []dbUser
	owner dbUser

	applicationData string
}

type dbUser struct {
	username string
	password string
	settings string
}

// InitDB initializes the database and establishes the connection to the database.
func (backend *Backend) InitDB(driverName string, connectionString string) {
	// Sets up the database connection with the driver and connection string. This does not initialize the connection to the database.
	var err error
	backend.db, err = sql.Open(driverName, connectionString)
	if err != nil {
		log.Fatal("Failed to initialize database: ", err)
	}
	// defer db.Close() closes the database connection when the function completes.
	defer backend.db.Close()

	// Initialize the connection to the database.
	err = backend.db.Ping()
	if err != nil {
		log.Fatal("Failed to establish a connection to the database: ", err)
	}
}

// Close function to explicitly close the connection to the database
func (backend *Backend) Close() error {
	err := backend.db.Close()
	if err != nil {
		return err
	}
	return nil
}

// Create Appends a struct to the database
func (backend *Backend) Create() error { return nil }

// Delete removes a given struct from the database.
func (backend *Backend) Delete() error { return nil }
