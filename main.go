package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"server/auth"
	geminiapi "server/geminiAPI"
	"server/model"
	"server/utils"
	ws "server/websocket"
	"strconv"
	"time"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"server/cloud"
)
func main() {
	// Use the SetServerAPIOptions() method to set the version of the Stable API on the client
	godotenv.Load()
	// Send a ping to confirm a successful connection
	client := utils.ConnectDB()
	redisClient := utils.ConnectRedis()
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{os.Getenv("FRONTEND_URL"), "http://localhost:8081","https://*.ngrok-free.app","http://localhost:5173"}, // Add your frontend origin
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With","ngrok-skip-browser-warning"},
		ExposeHeaders:    []string{"Content-Length","Set-Cookie"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	defer redisClient.Close()
	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()
	if err := client.Database("admin").RunCommand(context.TODO(), bson.D{{"ping", 1}}).Err(); err != nil {
		panic(err)
	}
	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")
	// Create a new WebSocket connection
	router.GET("/ws/:id", func(c *gin.Context) {
		ws.HandleWebSocket(c,client)
	})
	router.GET("/test/:userid", func(c *gin.Context) {
		id := "670aa7a22065dc72cb99f733"
		userid := c.Param("userid")
		objectId1, _ := primitive.ObjectIDFromHex(id)
		objectId2, _ := primitive.ObjectIDFromHex(userid)
		if err := model.CheckConversationUser( objectId2,objectId1, client); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})

	})
	router.GET("/ping", func(c *gin.Context) {
		if token, err := c.Request.Cookie("jwt_token"); err != nil {
			c.JSON(http.StatusOK, gin.H{"message": "no token"})
			return
		} else {
			if _, er := auth.VerifyJWT(token.Value); er != nil {
				c.JSON(http.StatusOK, gin.H{"message": "invalid token"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		}
	})
	router.POST("/registerEmail", func(c *gin.Context) {
		email := c.PostForm("email")
		if email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
			return
		}
		if err := model.RegisterNewEmail(email, client, redisClient); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	router.POST("/verify_register_OTP", func(c *gin.Context) {
		if !model.IsTokenNotValid(c, redisClient) {
			return
		}
		email := c.PostForm("email")
		otp := c.PostForm("otp")
		if email == "" || otp == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email and otp are required"})
			return
		}
		if err := model.VerifyOTP(email, otp, redisClient); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cookie := &http.Cookie{
			Name:     "register_token",
			Value:    utils.GenerateToken(email, redisClient),
			Expires:  time.Now().Add(15 * time.Minute),
			Path:	 "/",
			Domain:   "",
			MaxAge:   60*15,
			Secure: true,
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
		}
		http.SetCookie(c.Writer, cookie)
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	router.POST("/register", func(c *gin.Context) {
		if !model.IsTokenNotValid(c, redisClient) {
			return
		}
		var user model.User
		c.ShouldBind(&user)
		if user.Username == "" || user.Email == "" || user.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username, email, and password are required"})
			return
		}
		if cookie, err := c.Request.Cookie("register_token"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "please verify your email first"})
			return
		} else {
			if !utils.VerifyToken(user.Email, cookie.Value, redisClient) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "please verify your email first"})
				return
			}
		}
		cookie := &http.Cookie{
			Name:     "register_token",
			Value:    "",
			Expires:  time.Now().Add(-1 * time.Hour),
			Path:	 "/",
			Domain:   "",
			MaxAge:   -1,
			Secure: true,
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
		}
		http.SetCookie(c.Writer, cookie)
		if err := model.RegisterNewUser(&user, client, redisClient); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if token, er := auth.GenerateJWT(user.ID.Hex()); er != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": er.Error()})
			return
		} else {
			cookie := &http.Cookie{
				Name:     "jwt_token",
				Value:    token,
				Expires:  time.Now().Add(24 * time.Hour),
				Path:	 "/",
				Domain:   "",
				MaxAge:   86400,
				Secure: true,
				HttpOnly: true,
				SameSite: http.SameSiteNoneMode,
			}
			http.SetCookie(c.Writer, cookie)
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	router.POST("/login", func(c *gin.Context) {
		if !model.IsTokenNotValid(c, redisClient) {
			return
		}
		// if model.IsTokenValid(c, redisClient) {
		// 	c.JSON(http.StatusOK, gin.H{"message": "token is already valid"})
		// 	return
		// }
		var user model.User
		c.ShouldBind(&user)

		if userId, userName, err := model.Login(user.Email, user.Password, client); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		} else {
			if token, er := auth.GenerateJWT(userId); er != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": er.Error()})
				return
			} else {
				// setCookie(c,"jwt_token",token)
				cookie := &http.Cookie{
					Name:     "jwt_token",
					Value:    token,
					Expires:  time.Now().Add(24 * time.Hour),
					Path:	 "/",
					Domain:   "",
					MaxAge:   86400,
					Secure: true,
					HttpOnly: true,
					SameSite: http.SameSiteNoneMode,
				}
				http.SetCookie(c.Writer, cookie)
				// c.SetCookie("jwt_token", token, 60*60*24, "/", "", false, true)
			}
			
			c.JSON(http.StatusOK, gin.H{"message": "success", "userId": userId, "userEmail": user.Email, "userName": userName})
		}
	})
	router.GET("/conversations/:id", func(c *gin.Context) {
		if !model.IsTokenValid(c, redisClient) {
			return
		}
		var id int64
		var er error
		id, er = strconv.ParseInt(c.Param("id"), 10, 64)
		if er != nil || id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "bad page",
			})
			return
		}

		cookie, _ := c.Request.Cookie("jwt_token")
		token := cookie.Value
		payload, _ := auth.DecodeJWT(token)
		userID, err := primitive.ObjectIDFromHex(payload.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err,
			})
			return
		}
		if conversations, err := model.GetUserConversationsPage(userID, client, id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err,
			})
			return
		} else {
			c.JSON(http.StatusOK, gin.H{
				"message":       "success",
				"conversations": conversations,
			})
			return
		}
	})
	router.GET("/test/conversations", func(c *gin.Context) {
		type Test struct {
			ID          string    `json:"id"`
			Title       string    `json:"title"`
			LastMessage string    `json:"lastMessage"`
			UpdatedAt   time.Time `json:"updatedAt"`
			Unread      bool      `json:"unread"`
		}
		conversation1 := Test{
			ID:          "1",
			Title:       "First Conversation",
			LastMessage: "Last message in the conversation",
			UpdatedAt:   time.Date(2024, 3, 8, 12, 0, 0, 0, time.UTC),
			Unread:      false,
		}
		conversation2 := Test{
			ID:          "2",
			Title:       "Second Conversation",
			LastMessage: "Last message in the conversation",
			UpdatedAt:   time.Date(2024, 3, 9, 12, 0, 0, 0, time.UTC),
			Unread:      true,
		}
		conversation3 := Test{
			ID:          "3",
			Title:       "Thirsd Conversation",
			LastMessage: "Last message in the conversation3",
			UpdatedAt:   time.Date(2024, 3, 9, 18, 0, 0, 0, time.UTC),
			Unread:      false,
		}
		test := []Test{conversation1, conversation2, conversation3}

		c.JSON(http.StatusOK, gin.H{"list": test})
	})
	router.GET("/conversations", func(c *gin.Context) {
		if !model.IsTokenValid(c, redisClient) {
			return
		}
		cookie, _ := c.Request.Cookie("jwt_token")
		token := cookie.Value
		payload, _ := auth.DecodeJWT(token)
		userID, err := primitive.ObjectIDFromHex(payload.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err,
			})
			return
		}
		if conversations, err := model.GetUserConversations(userID, client); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err,
			})
			return
		} else {
			c.JSON(http.StatusOK, gin.H{
				"message":       "success",
				"conversations": conversations,
			})
			return
		}
	})
	router.GET("/conversation/:id", func(c *gin.Context) {
		if !model.IsTokenValid(c, redisClient) {
			return
		}
		id := c.Param("id")
		conversationID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": errors.New("invalid conversation id"),
			})
			return
		}
		cookie, _ := c.Request.Cookie("jwt_token")
		token := cookie.Value
		payload, _ := auth.DecodeJWT(token)
		userID, err := primitive.ObjectIDFromHex(payload.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err,
			})
			return
		}
		if conversation, err := model.GetOneConversation(conversationID, userID, client); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err,
			})
			return
		} else {
			c.JSON(http.StatusOK, gin.H{
				"message":      "success",
				"conversation": conversation,
			})
			return
		}
	})
	router.POST("/conversation/new", func(c *gin.Context) {
		if !model.IsTokenValid(c, redisClient) {
			return
		}
		mode := c.PostForm("mode")
		if mode != "1" && mode != "2" {
			mode = "1"
		}
		message := c.PostForm("message")
		cid := c.PostForm("cid")

		cookie, _ := c.Request.Cookie("jwt_token")
		token := cookie.Value
		payload, _ := auth.DecodeJWT(token)
		id, err := primitive.ObjectIDFromHex(payload.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err,
			})
			return
		}
		if id, er := model.AskNewConversation(id, message, client, mode,cid); er != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err,
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"cid": id.Hex(),
			})
		}
	})
	router.POST("/conversation/:id", func(c *gin.Context) {
		if !model.IsTokenValid(c, redisClient) {
			return
		}
		message := c.PostForm("message")
		id := c.Param("id")
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err,
			})
			return
		}
		cid := c.PostForm("cid")
		if err := model.AskInConversation(objectID, message, client,cid); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "success",
		})
	})
	router.GET("/api/get-signed-jwt", func(c *gin.Context) {
		if !model.IsTokenValid(c, redisClient) {
			return
		}
		cookie, _ := c.Request.Cookie("jwt_token")
		token := cookie.Value
		payload, _ := auth.DecodeJWT(token)
		if jwt, err := cloud.GetSignedJWT(payload.UserID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		} else {
			c.JSON(http.StatusOK, gin.H{"jwt": jwt})
		}
        // In production, you might want to validate the file type/size here
		
    })
	router.GET("/logout", func(c *gin.Context) {
		cookie := &http.Cookie{
			Name:     "jwt_token",
			Value:    "",
			Expires:  time.Now().Add(-1 * time.Second),
			Path:	 "/",
			Domain:   "",
			MaxAge:   -1,
			Secure: true,
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
		}
		http.SetCookie(c.Writer, cookie)
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	router.POST("/getTopic", func(c *gin.Context) {
		question := c.PostForm("question")
		if question == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "question is required"})
			return
		}
		if topic, err := geminiapi.GetTopic(question, false); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		} else {
			c.JSON(http.StatusOK, gin.H{"message": "success", "topic": topic})
		}
	})

	router.Run(":5000")
}
