package files

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sger/go-hashdir"
)

// File defines the data about a given file
type File struct {
	Name             string `json:"Name"`
	Path             string `json:"Path"`
	Size             int64  `json:"Size"`
	SizeSI           string `json:"SizeSI"`
	IsDir            bool   `json:"IsDir"`
	FileCount        int    `json:"FileCount"`
	Extension        string `json:"Extension"`
	ApplicaitionData string `json:"ApplicaitionData"`
	Hash             string `json:"Hash"`
}

// Files is a array of containing multiple instances of file.
type Files []File

/*
	converts bytes to kb/mb/gb/tb/pb/eb
	the last two are a bit overkill but future proofing? xD
*/
func (file *File) fileSizeToSI() {
	const unit = 1000
	if file.Size < unit {
		file.SizeSI = fmt.Sprintf("%d B", file.Size)
		return
	}
	div, exp := int64(unit), 0
	for n := file.Size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	file.SizeSI = fmt.Sprintf("%.1f %cB",
		float64(file.Size)/float64(div), "kMGTPE"[exp])
}

// creates sha256 has of the file.
func (file *File) createFileHash() (string, error) {
	hasher := sha256.New()
	bytes, err := ioutil.ReadFile(file.Path)
	hasher.Write(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// creates a string representation of the file structure.
func (file *File) toString() string {
	return "Name: " + file.Name +
		"\nSize: " + string(file.Size) +
		"\nPath: " + file.Path +
		"\nExtension: " + file.Extension +
		"\nIsDir: " + strconv.FormatBool(file.IsDir) +
		"\n ApplicaitionData: " + file.ApplicaitionData +
		"\n Hash: " + file.Hash
}

func (file *File) cleanPath(volumePath string) string {
	// Clean the path up
	Path := ""
	Path = strings.ReplaceAll(file.Path, "//", "/")
	Path = strings.ReplaceAll(Path, "/", "/")
	Path = strings.Replace(Path, volumePath, "", 1)
	return Path
}

// Volume contains the information about a given volume.
type Volume struct {
	Name string
	Path string
	Size int64
}

// Volumes is a array of containing multiple instances of Volume.
type Volumes []Volume

// WalkFolder takes in a path to a folder and returns a list of all the files inside the folder.
func (volume *Volume) WalkFolder(path string) (Files, error) {
	var files Files
	path = volume.cleanPath(path) + "/"
	filesInfo, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, info := range filesInfo {
		IsDir := info.IsDir()
		Size := info.Size()
		Path := path + info.Name()
		var file File
		if !IsDir {
			Extension := filepath.Ext(path + info.Name())
			Name := strings.Split(info.Name(), Extension)[0]
			file = File{Name: Name, Path: Path, Size: Size, IsDir: IsDir, Extension: Extension}
			file.fileSizeToSI()
			Hash, err := file.createFileHash()
			if err != nil {
				return nil, err
			}
			file.Hash = Hash
			file.Path = file.cleanPath(volume.Path)
		} else {
			Name := info.Name()
			fcount, err := ioutil.ReadDir(Path)
			if err != nil {
				return nil, err
			}
			file = File{Name: Name, Path: Path, Size: Size, IsDir: IsDir, FileCount: len(fcount)}
			Hash, err := hashdir.Create(file.Path, "sha256")
			if err != nil {
				return nil, err
			}
			file.Hash = Hash
			file.Path = file.cleanPath(volume.Path)
		}
		files = append(files, file)
	}
	return files, nil
}

// getFile gets a file form the given path.
func (volume *Volume) getFile(path string) ([]byte, error) {
	return nil, nil
}

// cleanPath cleans the path so the user cannot escape outside the specifiede volume
func (volume *Volume) cleanPath(path string) string {
	path = strings.Replace(path, "../", "", -1)
	return strings.Replace(path, "..", "", -1)
}
