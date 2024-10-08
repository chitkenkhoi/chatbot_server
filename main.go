package main

import (
	"context"
	"fmt"
	"server/model"
	"server/utils"

	"go.mongodb.org/mongo-driver/bson"
)

func main() {
	// Use the SetServerAPIOptions() method to set the version of the Stable API on the client

	// Send a ping to confirm a successful connection
	client := utils.ConnectDB()
	redisClient := utils.ConnectRedis()
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
	// user := model.User{
	// 	Username:       "newuser1",
	// 	Email:          "chitkenkhoi@gmail.com",
	// 	HashedPassword: "securepassword",
	// }
	// fmt.Println(model.RegisterNewUser(&user, client))
	// fmt.Println(utils.SendMail("lequangkhoim@gmail.com", "123456"))
	fmt.Println(model.RegisterNewEmail("lequangkhoim@gmail.com", client, redisClient))

}
