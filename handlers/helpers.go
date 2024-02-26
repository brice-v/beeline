package handlers

import (
	"beeline/db"
	"beeline/models"
	"fmt"
	"log"
	"regexp"
	"strconv"
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

func validateUsername(username string) error {
	unLen := len([]rune(username))
	if unLen < 3 || unLen > 255 {
		return fmt.Errorf("username length must be between 3 and 255, got %d", unLen)
	}
	if username == "admin" {
		return fmt.Errorf("invalid name")
	}
	reStr := `([a-zA-Z0-9]){3,255}`
	matched, err := regexp.MatchString(reStr, username)
	if err != nil {
		log.Fatal("Invalid Regex")
	}
	if !matched {
		return fmt.Errorf("invalid username `%s`, must match %s", username, reStr)
	}
	return nil
}

func validatePassword(password string) error {
	pwLen := len(password)
	if pwLen < 8 || pwLen > 255 {
		return fmt.Errorf("password length must be between 8 and 255, got %d", pwLen)
	}
	reStr := `.{8,255}`
	matched, err := regexp.MatchString(reStr, password)
	if err != nil {
		log.Fatal("Invalid Regex")
	}
	if !matched {
		return fmt.Errorf("invalid password must match %s", reStr)
	}
	return nil
}

// Return valid error for printing to the screen
func editUser(c *fiber.Ctx) error {
	userId := c.Params("id")
	newUn := c.FormValue("username")
	newFailedLoginAttempts := c.FormValue("failed_login_attempts")
	newPw := c.FormValue("new_password")
	newIsAdmin := c.FormValue("admin")
	id, err := strconv.ParseUint(userId, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid user id %s", userId)
	}
	newFailedLoginAttemptsNum, err := strconv.Atoi(newFailedLoginAttempts)
	if err != nil {
		return fmt.Errorf("invalid failed login attempts %s", newFailedLoginAttempts)
	}
	newIsAdminBool, err := strconv.ParseBool(newIsAdmin)
	if err != nil {
		return fmt.Errorf("invalid admin value %s", newIsAdmin)
	}
	// Get Current User so we can check if things changed
	dbc := getDB(c)
	currentUser := dbc.GetUser(id)

	if newUn != currentUser.Username {
		err = validateUsername(newUn)
		if err != nil {
			return fmt.Errorf("user id %s, %w", userId, err)
		}
		dbc.UpdateUserUsername(id, newUn)
	}
	if newPw != "" {
		err = validatePassword(newPw)
		if err != nil {
			return fmt.Errorf("user id %s, %w", userId, err)
		}
		dbc.UpdateUserPassword(id, newPw)
	}

	if newFailedLoginAttemptsNum != currentUser.FailedLoginAttempts {
		dbc.UpdateUserFailedLoginAttempts(id, newFailedLoginAttemptsNum)
	}

	if newIsAdminBool != currentUser.Admin {
		dbc.UpdateUserAdmin(id, newIsAdminBool)
	}

	return nil
}
