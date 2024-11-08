package websocket

import (
	"log"
	"net/http"
	"server/auth"
	"sync"
	"time"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/bson"
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
func HandleWebSocket(c *gin.Context,clientMongo *mongo.Client) {
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
	collection := clientMongo.Database("chatbot-server").Collection("conversation")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	chatIDObject, err := primitive.ObjectIDFromHex(chatID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat id"})
		return
	}
	userIDObject, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
		return
	}

	filter := bson.M{
		"_id":     chatIDObject,
		"user_id": userIDObject,
	}
	type ChatUser struct{
		ID        primitive.ObjectID   `bson:"_id,omitempty" json:"id,omitempty"`
		UserID    primitive.ObjectID   `bson:"user_id" json:"user_id"`
	}
	var chat ChatUser
	err1 := collection.FindOne(ctx, filter).Decode(&chat)
	if err1 != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err1.Error()})
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
