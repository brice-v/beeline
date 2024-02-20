package handlers

import (
	"beeline/db"
	"beeline/models"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

func getDB(c *fiber.Ctx) *db.DB {
	dbc, ok := c.Locals("db").(*db.DB)
	if !ok {
		log.Fatal("Database Connection not found in Locals")
	}
	return dbc
}

func setCookie(c *fiber.Ctx, key, value string) {
	cook := new(fiber.Cookie)
	cook.Name = key
	cook.Value = value
	cook.Expires = time.Now().Add(time.Hour)
	c.Cookie(cook)
}

func ValidateUser(c *fiber.Ctx, expectedUsername string) bool {
	currentUserAuthId := c.Cookies("authId")
	currentUsername := c.Cookies("username")
	if expectedUsername != currentUsername {
		return false
	}
	authId, ok := getDB(c).GetAuthId(currentUsername)
	if !ok {
		return false
	}
	return authId == currentUserAuthId
}

func checkAndGetCurrentUser(c *fiber.Ctx) (*models.User, bool) {
	unc := c.Cookies("username")
	if unc == "" {
		return nil, false
	}
	user, userFound := getDB(c).FindUser(unc)
	if !userFound {
		return nil, false
	}
	if !ValidateUser(c, unc) {
		return nil, false
	}
	return user, true
}
