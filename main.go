package main

import (
	"context"
	"fmt"
	"net/http"
	"server/model"
	"server/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"github.com/joho/godotenv"
	"server/auth"
)

func main() {
	// Use the SetServerAPIOptions() method to set the version of the Stable API on the client
	godotenv.Load()
	// Send a ping to confirm a successful connection
	client := utils.ConnectDB()
	redisClient := utils.ConnectRedis()
	router := gin.Default()



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



	router.GET("/ping", func(c *gin.Context) {
		if token,err := c.Request.Cookie("jwt_token");err!=nil{
			c.JSON(http.StatusOK, gin.H{"message": "no token"})
			return
		}else{
			if _,er := auth.VerifyJWT(token.Value);er!=nil{
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

		if user.Username == "" || user.Email == "" || user.HashedPassword == "" {
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
		if token,er := auth.GenerateJWT(user.ID.Hex());er!=nil{
			c.JSON(http.StatusBadRequest, gin.H{"error": er.Error()})
			return
		}else{
			c.SetCookie("jwt_token", token, 60*60*24, "/", "localhost", false, true)
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	router.POST("/login", func(c *gin.Context) {
		if model.IsTokenValid(c, redisClient) {
			c.JSON(http.StatusOK, gin.H{"message": "token is already valid"})
			return
		}
		email := c.PostForm("email")
		password := c.PostForm("password")
		if userId,err:=model.Login(email, password, client); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		} else {
			if token,er := auth.GenerateJWT(userId);er!=nil{
				c.JSON(http.StatusBadRequest, gin.H{"error": er.Error()})
				return
			}else{
				c.SetCookie("jwt_token", token, 60*60*24, "/", "localhost", false, true)
			}
			c.JSON(http.StatusOK, gin.H{"message": "success", "userId": userId})
		}
	})
	router.Run(":5000")
}
