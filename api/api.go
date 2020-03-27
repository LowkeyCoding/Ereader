// Package api Copyright 2020 LowkeyCoding
package api

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"./file"
	"github.com/go-http-utils/etag"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// V1 ................
type V1 struct {
	volume file.Volume
}

// getParameter returns a parameter if it exists else it returns a empty string a bool flag.
func (v1 *V1) getParameter(w http.ResponseWriter, r *http.Request, parameterName string) (string, bool) {
	parameter, ok := r.URL.Query()[parameterName]
	if !ok || len(parameter[0]) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "{\"ERROR\": \"Url Param '"+parameterName+"' is missing\"}")
		return "", false
	}
	return parameter[0], true
}

// cleanPath cleans a given path to eliminate users going out of a their given directory
func (v1 *V1) cleanPath(path string) string {
	path = strings.Replace(path, "../", "", -1)
	return strings.Replace(path, "..", "", -1)
}

// walkFolder returns the files inside a given directory in json format.
func (v1 *V1) walkFolder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	path, ok := v1.getParameter(w, r, "path")
	if !ok {
		return
	}
	files, err := v1.volume.WalkFolder(v1.cleanPath(path))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}
	fmt.Fprintf(w, string(files))
}

// getFile returns a file as a stream from a given path.
func (v1 *V1) getFile(w http.ResponseWriter, r *http.Request) {
	File, ok := v1.getParameter(w, r, "File")
	if !ok {
		return
	}
	openFile, err := os.Open(File)
	defer openFile.Close() //Close after function return
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	//File is found, create and send the correct headers

	//Get the Content-Type of the file
	//Create a buffer to store the header of the file in
	FileHeader := make([]byte, 512)
	//Copy the headers into the FileHeader buffer
	openFile.Read(FileHeader)
	//Get content type of file
	FileContentType := http.DetectContentType(FileHeader)

	//Get the file size
	FileStat, _ := openFile.Stat()                     //Get info from file
	FileSize := strconv.FormatInt(FileStat.Size(), 10) //Get file size as a string

	//Send the headers
	w.Header().Set("Content-Type", FileContentType)
	w.Header().Set("Content-Length", FileSize)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	//Send the file
	//We read 512 bytes from the file already, so we reset the offset back to 0
	openFile.Seek(0, 0)
	io.Copy(w, openFile) //'Copy' the file to the client
}

// API ................
type API struct {
	port           string
	scheme         string
	router         *mux.Router
	v1             *mux.Router
	currentVersion string
	oldVersions    []string
}

// optimizer handler, weak bool, level int.
func (api *API) optimizer(weak bool, level int) http.Handler {
	return etag.Handler(handlers.CompressHandlerLevel(api.router, level), weak)
}

// Init initializes the API.
func (api *API) Init(port string, scheme string) error {
	// currentVersion
	api.currentVersion = "v1"
	api.oldVersions = make([]string, 0)
	api.port = port
	api.scheme = scheme
	// Creates the main api path and NotFoundHandler
	api.router = mux.NewRouter().PathPrefix("/api").Subrouter()
	api.router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	// Initialize the v1 api router
	api.initV1()
	println("127.0.0.1:" + api.port)
	srv := &http.Server{
		Addr: "127.0.0.1:" + api.port,
		// timeout is set to prevet slowloris attack
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      api.optimizer(false, 6), // Pass our instance of gorilla/mux in.
	}
	if api.scheme == "http" {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal("ListenAndServe: ", err)
		}
	} else {
		log.Fatal("Failed to establish a connection using the givin scheme: ", errors.New(api.scheme))
		//err := http.ListenAndServeTLS(":"+api.port,certfile,keyfile, api.optimizer(false, 6))
	}
	return nil
}

// initV1 initializes api v1.
func (api *API) initV1() error {
	api.v1 = api.router.PathPrefix("/v1").Subrouter()
	V1 := V1{volume: file.Volume{Name: "C", Path: "./Volume", Size: 0}}
	api.v1.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")
	// Walksfolder get file and folder information from a given path
	api.v1.HandleFunc("/walkDir/", V1.walkFolder).Methods("GET")
	// GetFile gets the file from a given path
	api.v1.HandleFunc("/getFile/", V1.getFile).Methods("GET")
	api.v1.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	return nil
}
