package file

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	hashdir "github.com/sger/go-hashdir"
)

// File ....
type File struct {
	Name             string `json:"Name"`
	Path             string `json:"Path"`
	Size             int64  `json:"Size"`
	SizeSI           string `json:"SizeSI"`
	IsDir            bool   `json:"IsDir"`
	Extension        string `json:"Extension"`
	ApplicaitionData string `json:"ApplicaitionData"`
	Hash             string `json:"Hash"`
}

// Files ....
type Files []File

func (file *File) createFileHash() (string, error) {
	hasher := sha256.New()
	bytes, err := ioutil.ReadFile(file.Path)
	hasher.Write(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (file *File) fileSizeToSI() {
	const unit = 1000
	if file.Size < unit {
		file.SizeSI = fmt.Sprintf("%d B", file.Size)
	}
	div, exp := int64(unit), 0
	for n := file.Size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	file.SizeSI = fmt.Sprintf("%.1f %cB",
		float64(file.Size)/float64(div), "kMGTPE"[exp])
}

func (file *File) toString() string {
	return "Name: " + file.Name +
		"\nSize: " + string(file.Size) +
		"\nPath: " + file.Path +
		"\nExtension: " + file.Extension +
		"\nIsDir: " + strconv.FormatBool(file.IsDir) +
		"\n ApplicaitionData: " + file.ApplicaitionData +
		"\n Hash: " + file.Hash
}

// Volume ....
type Volume struct {
	Name string
	Path string
	Size int64
}

// WalkFolder returns a list of all files with json encoding of the given folder
func (volume *Volume) WalkFolder(path string) ([]byte, error) {
	var files Files
	path = volume.Path + path + "/"
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
		} else {
			Name := info.Name()
			file = File{Name: Name, Path: Path, Size: Size, IsDir: IsDir}
			Hash, err := hashdir.Create(file.Path, "sha256")
			if err != nil {
				return nil, err
			}
			file.Hash = Hash
		}
		files = append(files, file)
	}
	jsonFiles, err := json.Marshal(files)
	if err != nil {
		return nil, err
	}
	return jsonFiles, nil
}
func (volume *Volume) getFile(path string) ([]byte, error) {
	return nil, nil
}
