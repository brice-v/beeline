package main

import (
	"beeline/db"
	"beeline/handlers"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/encryptcookie"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/template/html/v2"
)

//go:embed views/*
var viewsStaticDir embed.FS

//go:embed public/*
var publicStaticDir embed.FS

const DB_NAME = "beeline.db?_pragma=journal_mode(WAL)"

type App struct {
	app *fiber.App
	dbc *db.DB
}

func NewApp() *App {
	engine := html.NewFileSystem(http.FS(viewsStaticDir), ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})
	a := &App{
		app: app,
	}
	a.setupMiddlewareAndDbc()
	a.setupRoutes()
	return a
}

func (a *App) Run() {
	go func() {
		if err := a.app.Listen(":5961"); err != nil {
			log.Panic("error while listening: " + err.Error())
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	fmt.Println("gracefully shutting down...")
	if err := a.app.Shutdown(); err != nil {
		log.Printf("FAILED to shutdown app, error: %s", err.Error())
	}

	fmt.Println("running cleanup tasks...")
	a.dbc.DeleteAllAuthIds()
	fmt.Println("shutdown complete!")
}

func (a *App) setupMiddlewareAndDbc() {
	a.app.Use(helmet.New())
	a.app.Use(encryptcookie.New(encryptcookie.Config{
		Key: encryptcookie.GenerateKey(),
	}))
	a.app.Use(compress.New())
	// use embedded public directory
	a.app.Use("/public", filesystem.New(filesystem.Config{
		Root:       http.FS(publicStaticDir),
		PathPrefix: "public",
	}))
	// use embedded favicon
	a.app.Use(favicon.New(favicon.Config{
		FileSystem: http.FS(publicStaticDir),
		File:       "public/favicon.ico",
	}))

	a.app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	dbc, err := db.NewAndMigrate(DB_NAME)
	if err != nil {
		log.Fatal(err)
	}
	a.dbc = dbc
	// store db connection in locals to be accessible by all reqs
	a.app.Use(func(c *fiber.Ctx) error {
		c.Locals("db", dbc)
		return c.Next()
	})
	// 1 req/s
	a.app.Use(limiter.New(limiter.Config{
		Expiration: time.Second,
	}))
}

func (a *App) setupRoutes() {
	a.app.Get("/", handlers.Index)
	a.app.Get("/signup", handlers.Signup)
	a.app.Get("/login", handlers.LoginUI)
	a.app.Get("/logout", handlers.GetLogout)
	a.app.Get("/user/:username", handlers.User)
	a.app.Get("/monitor", monitor.New(monitor.Config{
		Next: func(c *fiber.Ctx) bool {
			return !handlers.ValidateUser(c, "brice")
		},
	}))
	a.app.Get("/all", handlers.All)
	a.app.Get("/paste", handlers.Paste)
	a.app.Get("/my-pastes", handlers.MyPastes)
	a.app.Get("/paste/:id", handlers.GetPaste)

	a.app.Post("/paste", handlers.NewPaste)
	a.app.Post("/new-user", handlers.NewUser)
	a.app.Post("/login", handlers.Login)
	a.app.Post("/new-post", handlers.NewPost)
	a.app.Post("/logout", handlers.Logout)
	a.app.Post("/follow", handlers.Follow)

	a.app.Get("/chat", handlers.Chat)
	a.app.Post("/chat", handlers.ChatPost)
	a.app.Get("/chat/:room", handlers.ChatRoom)

	a.app.Get("/ws/chat/:room", handlers.WSChatRoom())
}

func main() {
	a := NewApp()
	a.Run()
}
