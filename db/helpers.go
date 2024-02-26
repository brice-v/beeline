package db

import (
	"log"

	"golang.org/x/crypto/bcrypt"
)

func generatePasswordHash(password string) string {
	// Generate "hash" to store from user password
	pwHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		// TODO: Properly handle error
		log.Fatal(err)
	}
	return string(pwHash)
}
