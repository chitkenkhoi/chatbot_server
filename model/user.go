package model

import (
	"context"
	"errors"
	"net/http"
	"server/auth"
	"server/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Username       string             `json:"username" bson:"username"`
	Email          string             `json:"email" bson:"email"`
	Password string             `json:"password" bson:"password"`
}

func RegisterNewEmail(email string, client *mongo.Client, redisClient *redis.Client) error {
	count, err := client.Database("chatbot-server").Collection("user").CountDocuments(context.TODO(), bson.M{"email": email})
	if err != nil {
		return errors.New("something is wrong please try again")
	}
	if count > 0 {
		return errors.New("email has been registered")
	}
	otp := utils.GenerateOTP()
	if err := utils.SendMail(email, otp); err != nil {
		return err
	}
	if err := redisClient.Set(context.TODO(), "otp_"+email, otp, 15*time.Minute).Err(); err != nil {
		return err
	}
	return nil
}
func VerifyOTP(email string, otp string, redisClient *redis.Client) error {
	value, err := redisClient.Get(context.TODO(), "otp_"+email).Result()
	if err != nil {
		return errors.New("otp has been expired")
	}
	if value != otp {
		return errors.New("otp is incorrect")
	}
	redisClient.Del(context.TODO(), "otp_"+email)
	return nil
}
func RegisterNewUser(user *User, client *mongo.Client, redisClient *redis.Client) error {
	db := client.Database("chatbot-server")
	collection := db.Collection("user")
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)
	result, err2 := collection.InsertOne(context.TODO(), user)
	if err2 != nil {
		return err2
	}
	user.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}
func Login(email, password string, client *mongo.Client) (string, error) {
	db := client.Database("chatbot-server")
	collection := db.Collection("user")
	var user User
	if err := collection.FindOne(context.TODO(), bson.M{"email": email}).Decode(&user); err != nil {
		return "", errors.New("email or password is incorrect")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", errors.New("email or password is incorrect")
	}
	return user.ID.Hex(), nil
}
func IsTokenValid(c *gin.Context, redisClient *redis.Client) bool{
	cookie, err := c.Request.Cookie("jwt_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": errors.New("not authenticate"),
		})
		return false
	}
	token := cookie.Value
	claims, err := auth.VerifyJWT(token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": errors.New("token expired"),
		})
		return	false
	}
	if _, err := redisClient.Get(context.TODO(), "blacklist_"+claims.UserID).Result(); err == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": errors.New("token has been blacklisted"),
		})
		return 		false
	}
	return	true
}
func IsTokenNotValid(c *gin.Context, redisClient *redis.Client) bool {
	cookie, err := c.Request.Cookie("jwt_token")
	if err != nil {
		return	true
	}
	token := cookie.Value
	claims, err := auth.VerifyJWT(token)
	if err != nil {
		return true
	}
	if _, err := redisClient.Get(context.TODO(), "blacklist_"+claims.UserID).Result(); err == nil {
		return true
	}

	c.JSON(200,gin.H{
		"message":"already login",
	})
	return false
}