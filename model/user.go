package model

import (
	"context"
	"errors"
	"fmt"
	"server/utils"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	Username       string             `bson:"username"`
	Email          string             `bson:"email"`
	HashedPassword string             `bson:"hashedPassword"`
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
	if err := redisClient.Set(context.TODO(), "otp_"+email, otp, 5*time.Minute).Err(); err != nil {
		return err
	}
	return nil
}
func RegisterNewUser(user *User, client *mongo.Client) error {
	db := client.Database("chatbot-server")
	collection := db.Collection("user")
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.HashedPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.HashedPassword = string(hashedPassword)
	result, err2 := collection.InsertOne(context.TODO(), user)
	if err2 != nil {
		return err2
	}
	fmt.Println("Inserted document ID:", result.InsertedID)
	return nil
}
