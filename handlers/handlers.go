package handlers

import (
	"beeline/models"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const MaxFailedLoginAttempts = 3

func Monitor() func(*fiber.Ctx) error {
	return monitor.New(monitor.Config{
		Next: func(c *fiber.Ctx) bool {
			user, isValid := checkAndGetCurrentUser(c)
			return !isValid || !user.IsAdmin()
		},
	})
}

func Index(c *fiber.Ctx) error {
	user, isValid := checkAndGetCurrentUser(c)
	if !isValid {
		return c.Redirect("/login")
	}
	posts := getDB(c).GetPosts(user)
	return c.Render("views/home", fiber.Map{
		"Username": user.Username,
		"Posts":    posts,
		"IsAdmin":  user.IsAdmin(),
	})
}

func Signup(c *fiber.Ctx) error {
	user, isValid := checkAndGetCurrentUser(c)
	if !isValid || !user.IsAdmin() {
		return c.SendStatus(fiber.StatusForbidden)
	}
	return c.Render("views/signup", "")
}

func NewUser(c *fiber.Ctx) error {
	user, isValid := checkAndGetCurrentUser(c)
	if !isValid || !user.IsAdmin() {
		return c.SendStatus(fiber.StatusForbidden)
	}
	un := c.FormValue("username")
	if un == "admin" {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	err := validateUsername(un)
	if err != nil {
		return c.Render("views/signup", fiber.Map{
			"Error": err.Error(),
		})
	}
	pw := c.FormValue("password")
	err = validatePassword(pw)
	if err != nil {
		return c.Render("views/signup", fiber.Map{
			"Error": err.Error(),
		})
	}
	dbc := getDB(c)
	if _, ok := dbc.FindUser(un); ok {
		errorString := fmt.Sprintf("Username '%s' already exists!", un)
		return c.Render("views/signup", fiber.Map{
			"Error": errorString,
		})
	}
	if dbc.TotalUserCount() > 100 {
		return c.SendString("Max users of 100 reached")
	}
	dbc.CreateUser(un, pw, false)
	if user.IsAdmin() {
		return c.Redirect("/")
	}
	// This path is only really applicable if we allow random signups
	authId := uuid.New().String()
	getDB(c).SetAuthId(un, authId)
	setCookie(c, "username", un)
	setCookie(c, "authId", authId)
	return c.Redirect("/")
}

func LoginUI(c *fiber.Ctx) error {
	return c.Render("views/login", "")
}

func Login(c *fiber.Ctx) error {
	un := c.FormValue("username")
	pw := c.FormValue("password")
	dbc := getDB(c)
	user, ok := dbc.FindUser(un)
	if !ok {
		return c.Render("views/login", fiber.Map{
			"Error": "User Not Found!",
		})
	}

	if user.FailedLoginAttempts > MaxFailedLoginAttempts {
		log.Printf("user `%s` attempting to login with failed login attempts > MaxFailedLoginAttempts", user)
		return c.Render("views/login", fiber.Map{
			"Error": "Too many failed logins! Please contact server admin.",
		})
	}
	// Comparing the password with the hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(pw)); err != nil {
		dbc.IncrementFailedLoginAttempts(un)
		return c.Render("views/login", fiber.Map{
			"Error": "Invalid Credentials!",
		})
	} else {
		dbc.ResetFailedLoginAttempts(un)
	}
	authId := uuid.New().String()
	getDB(c).SetAuthId(un, authId)
	setCookie(c, "username", un)
	setCookie(c, "authId", authId)
	return c.Redirect("/")
}

func User(c *fiber.Ctx) error {
	// username were getting to
	un := c.Params("username")
	currentUser, isValid := checkAndGetCurrentUser(c)
	if !isValid {
		return c.Redirect("/login")
	}
	dbc := getDB(c)
	user, ok := dbc.FindUser(un)
	if !ok {
		return c.SendString("User '" + un + "' not found!")
	}
	posts := dbc.GetSingleUsersPosts(user)
	return c.Render("views/user", fiber.Map{
		"Username":           un,
		"IsUsernameLoggedIn": un == currentUser.Username,
		"FollowerUsername":   currentUser.Username,
		"Posts":              posts,
		"IsNotFollowing":     !dbc.IsUserFollowing(un, currentUser.Username),
	})
}

func Users(c *fiber.Ctx) error {
	user, isValid := checkAndGetCurrentUser(c)
	if !isValid {
		return c.Redirect("/login")
	}
	if !user.IsAdmin() {
		return c.SendStatus(fiber.StatusForbidden)
	}

	// Render Users Page where as admin I can reset failed login attempts or update passwords
	allUsers := getDB(c).GetAllUsers()
	return c.Render("views/users", fiber.Map{"IsAdmin": user.IsAdmin(), "Username": user.Username, "Users": allUsers})
}

func EditUser(c *fiber.Ctx) error {
	user, isValid := checkAndGetCurrentUser(c)
	if !isValid {
		return c.Redirect("/login")
	}
	if !user.IsAdmin() {
		return c.SendStatus(fiber.StatusForbidden)
	}
	userId := c.Params("id")
	err := editUser(c)
	if err != nil {
		// If theres an error validating the edit we need to return
		allUsers := getDB(c).GetAllUsers()
		return c.Render("views/users", fiber.Map{"IsAdmin": user.IsAdmin(), "Username": user.Username, "Users": allUsers, "Error": err.Error()})
	}
	allUsers := getDB(c).GetAllUsers()
	successStr := fmt.Sprintf("Successfully Update User ID %s", userId)
	return c.Render("views/users", fiber.Map{"IsAdmin": user.IsAdmin(), "Username": user.Username, "Users": allUsers, "Success": successStr})
}

func NewPost(c *fiber.Ctx) error {
	m := c.FormValue("message")
	un := c.FormValue("username")
	if !validateUser(c, un) {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	post := &models.Post{
		Message:   m,
		Timestamp: time.Now(),
		Username:  un,
	}
	getDB(c).NewPost(post)
	return c.Redirect("/")
}

func Logout(c *fiber.Ctx) error {
	un := c.FormValue("username")
	c.ClearCookie("username", "authId")
	getDB(c).DeleteAuthId(un)
	return c.Redirect("/")
}

func GetLogout(c *fiber.Ctx) error {
	return Logout(c)
}

func Follow(c *fiber.Ctx) error {
	un := c.FormValue("username")
	f := c.FormValue("follower")
	getDB(c).FollowUser(un, f)
	return c.Redirect("/user/" + un)
}

func All(c *fiber.Ctx) error {
	posts := getDB(c).GetAllPosts()
	return c.Render("views/all", fiber.Map{
		"Posts": posts,
	})
}

func NewPaste(c *fiber.Ctx) error {
	user, isValid := checkAndGetCurrentUser(c)
	if !isValid {
		return c.Redirect("/login")
	}
	title := c.FormValue("title")
	text := c.FormValue("text")
	un := c.FormValue("username")
	if !validateUser(c, un) {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	p := &models.Paste{
		Title:    title,
		Text:     text,
		Username: un,
	}
	err := p.Validate()
	if err != nil {
		log.Printf("POST /paste error: %s", err.Error())
		return c.Render("views/paste", fiber.Map{
			"Error": err.Error(),
		})
	}
	getDB(c).NewPaste(p)
	return c.Render("views/paste-ro", fiber.Map{
		"Username": un,
		"Title":    title,
		"Text":     text,
		"Id":       p.ID,
		"IsAdmin":  user.IsAdmin(),
	})
}

func Paste(c *fiber.Ctx) error {
	user, isValid := checkAndGetCurrentUser(c)
	if !isValid {
		return c.Redirect("/login")
	}
	return c.Render("views/paste", fiber.Map{"IsAdmin": user.IsAdmin(), "Username": user.Username})
}

func GetPaste(c *fiber.Ctx) error {
	user, isValid := checkAndGetCurrentUser(c)
	if !isValid {
		return c.Redirect("/login")
	}
	sid := c.Params("id")
	id, err := strconv.ParseUint(sid, 10, 64)
	if err != nil {
		log.Printf("GetPaste: Params(id) was not uint, error: %s", err.Error())
		return c.Redirect("/my-pastes")
	}
	paste, ok := getDB(c).GetPaste(user, id)
	if !ok {
		log.Printf("GetPaste: Paste not found")
		return c.Redirect("/my-pastes")
	}
	return c.Render("views/paste-ro", fiber.Map{
		"Username": user.Username,
		"Title":    paste.Title,
		"Text":     paste.Text,
		"Id":       paste.ID,
		"IsAdmin":  user.IsAdmin(),
	})
}

func MyPastes(c *fiber.Ctx) error {
	user, isValid := checkAndGetCurrentUser(c)
	if !isValid {
		return c.Redirect("/login")
	}

	pastes := getDB(c).GetAllPastes(user)
	return c.Render("views/my-pastes", fiber.Map{
		"Username": user.Username,
		"Pastes":   pastes,
		"IsAdmin":  user.IsAdmin(),
	})
}

func Chat(c *fiber.Ctx) error {
	user, isValid := checkAndGetCurrentUser(c)
	if !isValid {
		return c.Redirect("/login")
	}
	return c.Render("views/chat", fiber.Map{
		"Username": user.Username,
		"IsAdmin":  user.IsAdmin(),
	})
}

func ChatPost(c *fiber.Ctx) error {
	_, isValid := checkAndGetCurrentUser(c)
	if !isValid {
		return c.Redirect("/login")
	}
	room := c.FormValue("room")
	if room == "" {
		return c.Redirect("/chat")
	}

	return c.Redirect("/chat/" + url.PathEscape(room))
}

func ChatRoom(c *fiber.Ctx) error {
	user, isValid := checkAndGetCurrentUser(c)
	if !isValid {
		return c.Redirect("/login")
	}
	room := c.Params("room")
	if room == "" {
		return c.Redirect("/chat")
	}
	return c.Render("views/chatroom", fiber.Map{
		"Room":     room,
		"Username": user.Username,
		"IsAdmin":  user.IsAdmin(),
	})
}

func WSChatRoom() func(*fiber.Ctx) error {
	return websocket.New(func(c *websocket.Conn) {
		log.Println(c.Params("room")) // 123

		user, isValid := checkAndGetCurrentUserWS(c)
		if !isValid {
			c.Close()
			return
		}
		var (
			mt  int
			msg []byte
			err error
		)
		for {
			if mt, msg, err = c.ReadMessage(); err != nil {
				log.Println("read:", err)
				break
			}
			if mt != websocket.TextMessage {
				c.Close()
				return
			}
			var cm models.ChatMessage
			if err := json.Unmarshal(msg, &cm); err != nil {
				log.Printf("json unmarshal: %s", err.Error())
				break
			}
			if len([]rune(cm.Message)) < 3 || len([]rune(cm.Message)) > 255 || cm.Username != user.Username {
				continue
			}
			cm.Timestamp = time.Now()
			log.Printf("mt = %d, recv: %s, cm = %s", mt, msg, cm)

			if err = c.WriteMessage(mt, cm.ToTextMessage()); err != nil {
				log.Println("write:", err)
				break
			}
		}
	})
}
