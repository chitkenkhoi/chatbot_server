package websocket
import(
	"net/http"
	"github.com/gorilla/websocket"
	"sync"
	"github.com/gin-gonic/gin"
	"log"
	"server/auth"
)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now. In production, you should restrict this.
	},
}
type Client struct {
	conn   *websocket.Conn
	userID string
}
var clients = make(map[string]*Client)
var clientsMutex sync.Mutex
func TestWebSocket() {
	// This is a test function that does nothing.
}
func HandleWebSocket(c *gin.Context) {
	var token string
	var userID string
	if cookie,err := c.Request.Cookie("jwt_token");err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No cookie found"})
		return
	}else{
		token = cookie.Value
	}
	if _, err := auth.VerifyJWT(token); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}
	if payload, err := auth.DecodeJWT(token);err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err})
		return
	}else{
		userID = payload.UserID
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}
	defer conn.Close()
	client := &Client{conn: conn, userID: userID}
	clientsMutex.Lock()
	clients[userID] = client
	clientsMutex.Unlock()
	defer func() {
		clientsMutex.Lock()
		delete(clients, userID)
		clientsMutex.Unlock()
	}()
	for {
		// Keep the connection open
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}
	}
}
func BroadcastToken(userID,token string) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	if userID == "" {
		log.Println("No user ID provided")
		return
	}
	client, exists := clients[userID]
	if !exists {
		log.Printf("User %s not found\n", userID)
		return
	}
	err := client.conn.WriteMessage(websocket.TextMessage, []byte(token))
	if err != nil {
		log.Printf("Error sending message to user %s: %v\n", userID, err)
		delete(clients, userID)
	}
}