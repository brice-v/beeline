package handlers

import (
	"beeline/db"
	"beeline/models"
	"log"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func getDB(c *fiber.Ctx) *db.DB {
	dbc, ok := c.Locals("db").(*db.DB)
	if !ok {
		log.Fatal("Database Connection not found in Locals")
	}
	return dbc
}
func getDBWS(c *websocket.Conn) *db.DB {
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

func validateUser(c *fiber.Ctx, expectedUsername string) bool {
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

func validateUserWS(c *websocket.Conn, expectedUsername string) bool {
	currentUserAuthId := c.Cookies("authId")
	currentUsername := c.Cookies("username")
	if expectedUsername != currentUsername {
		return false
	}
	authId, ok := getDBWS(c).GetAuthId(currentUsername)
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
	if !validateUser(c, unc) {
		return nil, false
	}
	return user, true
}

func checkAndGetCurrentUserWS(c *websocket.Conn) (*models.User, bool) {
	unc := c.Cookies("username")
	if unc == "" {
		return nil, false
	}
	user, userFound := getDBWS(c).FindUser(unc)
	if !userFound {
		return nil, false
	}
	if !validateUserWS(c, unc) {
		return nil, false
	}
	return user, true
}
