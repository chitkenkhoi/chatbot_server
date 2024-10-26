package websocket

import (
	"log"
	"net/http"
	"server/auth"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now. In production, you should restrict this.
	},
}

type Client struct {
	conn *websocket.Conn
	id   string
}

var clients = make(map[string]*Client)
var clientsMutex sync.Mutex

func TestWebSocket() {
	// This is a test function that does nothing.
}
func HandleWebSocket(c *gin.Context) {
	var token string
	var userID string
	chatID := c.Param("id")
	if chatID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No chat id found",
		})
		return // Added return statement here
	}

	// Cookie validation
	cookie, err := c.Request.Cookie("jwt_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No cookie found"})
		return
	}
	token = cookie.Value

	// JWT validation
	if _, err := auth.VerifyJWT(token); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	payload, err := auth.DecodeJWT(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err})
		return
	}
	userID = payload.UserID

	// Upgrade connection
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}

	// Create client and add to clients map
	id := userID + ":" + chatID
	client := &Client{
		conn: conn,
		id:   id,
	}

	clientsMutex.Lock()
	clients[id] = client
	clientsMutex.Unlock()

	// Ensure cleanup
	defer func() {
		conn.Close()
		clientsMutex.Lock()
		delete(clients, id)
		clientsMutex.Unlock()
	}()

	// Add ping/pong handlers to detect disconnection more reliably
	conn.SetPingHandler(func(string) error {
		return conn.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(time.Second))
	})

	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start a ping ticker
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second)); err != nil {
					return
				}
			}
		}
	}()

	for {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Unexpected close error: %v", err)
			}
			break
		}
	}
}
func BroadcastToken(userID, chatID, token string) {
	if userID == "" {
		log.Println("No user ID provided")
		return
	}
	if chatID == "" {
		log.Println("No chat ID provided")
		return
	}

	clientID := userID + ":" + chatID

	// Create a timeout channel
	timeout := time.After(50 * time.Second) // Adjust timeout duration as needed

	// Create a ticker for polling
	ticker := time.NewTicker(100 * time.Millisecond) // Poll every 100ms
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			log.Printf("Timeout waiting for client connection: %s\n", clientID)
			return

		case <-ticker.C:
			clientsMutex.Lock()
			client, exists := clients[clientID]
			if exists {
				err := client.conn.WriteMessage(websocket.TextMessage, []byte(token))
				if err != nil {
					log.Printf("Error sending message to user %s: %v\n", clientID, err)
					delete(clients, clientID)
				}
				clientsMutex.Unlock()
				return
			}
			clientsMutex.Unlock()
		}
	}
}
