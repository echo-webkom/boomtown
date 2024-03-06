package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type message struct {
	RegisterCount int `json:"registerCount"`
	WaitlistCount int `json:"waitlistCount"`
}

type client struct {
	HappeningID string
}

var clients = make(map[*websocket.Conn]client)
var register = make(chan struct {
	*websocket.Conn
	ID string
})
var broadcast = make(chan string)
var unregister = make(chan *websocket.Conn)

var Db *gorm.DB

func initDB() {
	db, err := gorm.Open(postgres.Open(os.Getenv("DB_URL")), &gorm.Config{
		// Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Println("Failed to connect to database")
		log.Fatal(err)
		os.Exit(2)
	}

	log.Println("Connected to database")
	// db.Logger = db.Logger.LogMode(logger.Info)

	// db.Exec("DROP TABLE IF EXISTS users CASCADE")

	// log.Println("Migrating the schema...")
	// db.AutoMigrate(&User{})

	Db = db

}

func runHub() {
	for {
		select {
		case connection := <-register:
			clients[connection.Conn] = client{
				HappeningID: connection.ID,
			}
			log.Println("connection registered")

		case id := <-broadcast:
			log.Println("message id:", id)

			message := message{
				RegisterCount: 2,
				WaitlistCount: 1,
			}

			jsonMessage, err := json.Marshal(message)
			if err != nil {
				log.Println("json error:", err)
			}

			// TODO GET COUNT FROM DB
			// Db.Exec("SELECT COUNT(*) FROM registrations WHERE happening_id = ?", id)
			result, err := Db.Raw("SELECT COUNT(*) FROM registrations WHERE happening_id = ?", id).Rows()
			if err != nil {
				log.Println("DB error:", err)
			}

			var count int
			for result.Next() {
				result.Scan(&count)
			}

			log.Println("!!! count:", count)

			for connection := range clients {
				if clients[connection].HappeningID == id {
					if err := connection.WriteMessage(websocket.TextMessage, []byte(jsonMessage)); err != nil {
						log.Println("write error:", err)

						unregister <- connection
						connection.WriteMessage(websocket.CloseMessage, []byte{})
						connection.Close()
					}
				}
			}

		case connection := <-unregister:
			delete(clients, connection)

			log.Println("connection unregistered")
		}
	}
}

func main() {
	initDB()
	app := fiber.New()

	app.Use("/ws", func(c *fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	go runHub()

	app.Post("/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		broadcast <- id
		return c.SendStatus(fiber.StatusOK)
	})

	app.Get("/ws/:id", websocket.New(func(c *websocket.Conn) {
		id := c.Params("id")
		defer func() {
			unregister <- c
			c.Close()
		}()

		client := struct {
			*websocket.Conn
			ID string
		}{c, id}

		register <- client
	}))

	addr := flag.String("addr", ":8080", "http service address")
	flag.Parse()
	app.Listen(*addr)
}
