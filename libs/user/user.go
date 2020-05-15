package user

import (
	Files "../files"
)

// < ----- User ----- >

// User is a struct representing a user in the database
type User struct {
	ID             string             `json:"ID"`
	Username       string             `json:"Username"`
	Password       string             `json:"Password"`
	ProfilePicture string             `json:"ProfilePicture"`
	FileSettings   Files.FileSettings `json:"FileSettings"`
}
