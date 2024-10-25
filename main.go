package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"server/auth"
	chatbotapi "server/chatbotAPI"
	"server/model"
	"server/utils"
	ws "server/websocket"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func main() {
	// Use the SetServerAPIOptions() method to set the version of the Stable API on the client
	godotenv.Load()
	// Send a ping to confirm a successful connection
	client := utils.ConnectDB()
	redisClient := utils.ConnectRedis()
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8081"}, // Add your frontend origin
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
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
	router.GET("/ws", func(c *gin.Context) {
		ws.HandleWebSocket(c)
	})
	router.GET("/test", func(c *gin.Context) {
		for token := range chatbotapi.GetStreamingResponseFromModelAPIDemo() {
			ws.BroadcastToken("", token)
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
		c.SetCookie("register_token", utils.GenerateToken(email, redisClient), 60*15, "/", "localhost", false, true)
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	router.POST("/register", func(c *gin.Context) {
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
		c.SetCookie("register_token", "", -1, "/", "localhost", false, true)
		if err := model.RegisterNewUser(&user, client, redisClient); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if token, er := auth.GenerateJWT(user.ID.Hex()); er != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": er.Error()})
			return
		} else {
			c.SetCookie("jwt_token", token, 60*60*24, "/", "localhost", false, true)
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

		if userId, err := model.Login(user.Email, user.Password, client); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		} else {
			if token, er := auth.GenerateJWT(userId); er != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": er.Error()})
				return
			} else {
				c.SetCookie("jwt_token", token, 60*60*24, "/", "localhost", false, true)
			}
			c.JSON(http.StatusOK, gin.H{"message": "success", "userId": userId})
		}
	})
	router.GET("/conversations", func(c *gin.Context) {
		if !model.IsTokenValid(c, redisClient){
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
		message := c.PostForm("message")
		fmt.Println(message)
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
		model.AskNewConversation(id, message, client)
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
		model.AskInConversation(objectID, message, client)
	})
	router.Run(":5000")
}
