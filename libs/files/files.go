package files

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sger/go-hashdir"
)

// File defines the data about a given file
type File struct {
	Name        string      `json:"Name"`
	Path        string      `json:"Path"`
	Size        int64       `json:"Size"`
	SizeSI      string      `json:"SizeSI"`
	IsDir       bool        `json:"IsDir"`
	FileCount   int         `json:"FileCount"`
	Extension   string      `json:"Extension"`
	FileSetting FileSetting `json:"FileSetting"`
	Hash        string      `json:"Hash"`
	User        string      `json:"User"`
}

// Files is a array of containing multiple instances of file.
type Files []File

// AddFileSetting adds a file setting
func (files Files) AddFileSetting(settingsMap map[string]FileSetting) Files {
	for i := range files {
		files[i].FileSetting = settingsMap[files[i].Extension]
		if files[i].Extension != "" {
			files[i].FileSetting.Icon = files[i].Extension[1:]
		}
	}
	return files
}

// ToJSON converts a given file to JSON
func (files *Files) ToJSON() (string, error) {
	b, err := json.Marshal(&files)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

/*
FileSizeToSI converts bytes to kb/mb/gb/tb/pb/eb
the last two are a bit overkill but future proofing? xD
*/
func (file *File) FileSizeToSI() {
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

// CreateFileHash creates a sha256 hash of the given file.
func (file *File) CreateFileHash() (string, error) {
	hasher := sha256.New()
	bytes, err := ioutil.ReadFile(file.Path)
	hasher.Write(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// ToString creates a string representation of the file structure.
func (file *File) ToString() string {
	return "Name: " + file.Name +
		"\nSize: " + string(file.Size) +
		"\nPath: " + file.Path +
		"\nExtension: " + file.Extension +
		"\nIsDir: " + strconv.FormatBool(file.IsDir) +
		"\nFileCount: " + strconv.Itoa(file.FileCount) +
		"\nHash: " + file.Hash
}

// CleanPath Cleans the path by removing redundant slashes
func (file *File) CleanPath(volumePath string) string {
	// Clean the path up
	Path := ""
	Path = strings.ReplaceAll(file.Path, "//", "/")
	Path = strings.ReplaceAll(Path, "/", "/")
	Path = strings.Replace(Path, volumePath, "", 1)
	return Path
}

// < ----- Settings ----- >

// FileSetting is a struct representing the settings for a given file type.
type FileSetting struct {
	ID              string `json:"ID" db:"ID"`
	Username        string `json:"Username" db:"Username"`
	Extension       string `json:"Extension" db:"Extension"` // (.)type
	Icon            string `json:"Icon" db:"Icon"`
	ApplicationLink string `json:"ApplicationLink" db:"ApplicationLink"`
}

// ToJSON converts a given FileSetting to a JSON formattet string
func (setting *FileSetting) ToJSON() (string, error) {
	b, err := json.Marshal(&setting)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ToStruct converts a JSON formattet string into the structure
func (setting *FileSetting) ToStruct(jsonData string) error {
	jsonMap := FileSetting{}
	err := json.Unmarshal([]byte(jsonData), &jsonMap)
	if err != nil {
		return err
	}
	*setting = jsonMap
	return nil
}

// FileSettings is a array of multiple instances of FileSetting.
// This is used to keep track of a induvidual users file settings.
type FileSettings []FileSetting

// ToJSON converts a given set of FileSettings it will be converted to a JSON formattet string
func (settings *FileSettings) ToJSON() (string, error) {
	b, err := json.Marshal(&settings)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ToStruct converts a JSON formattet string into the structure
func (settings *FileSettings) ToStruct(jsonData string) error {
	jsonMap := []FileSetting{}
	err := json.Unmarshal([]byte(jsonData), &jsonMap)
	if err != nil {
		return err
	}
	*settings = jsonMap
	return nil
}

// ToMap converts the FileSettings struct to a map with the setting extension as a key the individual settings.
func (settings *FileSettings) ToMap() map[string]FileSetting {
	settingsMap := make(map[string]FileSetting)
	for _, setting := range *settings {
		settingsMap[setting.Extension] = setting
	}
	return settingsMap
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
	path = volume.CleanPath(path) + "/"
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
			file.FileSizeToSI()
			Hash, err := file.CreateFileHash()
			if err != nil {
				return nil, err
			}
			file.Hash = Hash
			file.Path = file.CleanPath(volume.Path)
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
			file.Path = file.CleanPath(volume.Path)
		}
		files = append(files, file)
	}
	return files, nil
}

// CleanPath cleans the path so the user cannot escape outside the specifiede volume
func (volume *Volume) CleanPath(path string) string {
	path = strings.Replace(path, "../", "", -1)
	return strings.Replace(path, "..", "", -1)
}
