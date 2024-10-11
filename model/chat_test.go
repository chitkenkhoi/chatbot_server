package model

import (
	"fmt"
	"testing"
	"server/utils"
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCreateConversation(t *testing.T){
	fmt.Println("TestCreateConversation")
	id :="6706b4c98fc002acdb37afdc"
	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
        // Handle the error (invalid ObjectId format, etc.)
        panic(err)
    }
	client:= utils.ConnectDB()
	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()
	fmt.Println(CreateConversation(NewConversation(objectId, "Hello World"),client))
}