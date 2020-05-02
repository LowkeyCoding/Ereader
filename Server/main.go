package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// "Signin" and "Signup" are handler that we will implement
	server := &Server{}
	http.HandleFunc("/signin", server.signin)
	http.HandleFunc("/signup", server.signup)
	server.initDB()
	// start the server on port 8000
	log.Fatal(http.ListenAndServe(":8000", nil))
}

// < ----- Server ----- >
type Server struct {
	db *sql.DB
}

// < ----- initDB ----- >
func (server *Server) initDB() {
	// Connect to the postgres db
	//you might have to change the connection string to add your database credentials
	var err error
	server.db, err = sql.Open("sqlite3", "./Users.db")
	if err != nil {
		panic(err)
	}
	// Setup the database table if it doesn't exist'
	statement, err := server.db.Prepare(`
		CREATE TABLE IF NOT EXISTS users(
			ID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			Username TEXT,
			Password TEXT
		);
	`)
	if err != nil {
		panic(err)
	}
	statement.Exec()
}

// < ----- User ----- >
type User struct {
	ID       string `json: "ID" db: "ID"`
	Password string `json: "password" db: "password"`
	Username string `json: "password" db: "username"`
}

// < ----- Signup ----- >
func (server *Server) signup(w http.ResponseWriter, r *http.Request) {
	user := &User{}
	err := json.NewDecoder(r.Body).Decode(user)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	userExists := server.getUserByUsername(user.Username)
	if userExists.Username == user.Username {
		w.WriteHeader(http.StatusForbidden)
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 8)
	err = server.insertUser(user.Username, string(hashedPassword))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// < ----- Signin ----- >
func (server *Server) signin(w http.ResponseWriter, r *http.Request) {
	user := &User{}
	err := json.NewDecoder(r.Body).Decode(user)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	storedUser := server.getUserByUsername(user.Username)
	if storedUser.ID == sql.ErrNoRows.Error() {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(storedUser.Password), []byte(user.Password))
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
	}
	w.WriteHeader(http.StatusOK)
}

// < ----- insertUser ----- >
func (server *Server) insertUser(username string, password string) error {
	statement, _ := server.db.Prepare(`
		INSERT INTO users (username, password) values (?,?)
	`)
	_, err := statement.Exec(username, password)
	if err != nil {
		return err
	}
	return nil
}

// < ----- getUserByUsername ----- >
func (server *Server) getUserByUsername(username string) User {
	result := server.db.QueryRow("select * from users where username=$1", username)
	user := User{}
	result.Scan(&user.ID, &user.Username, &user.Password)
	return user
}
