package model

import (
	"context"
	"errors"
	"fmt"
	chatbotapi "server/chatbotAPI"
	"server/utils"
	ws "server/websocket"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Message struct {
	Sender    string    `bson:"sender" json:"sender"`
	Content   string    `bson:"content" json:"content"`
	Timestamp time.Time `bson:"timestamp" json:"timestamp"`
	Token     int       `bson:"token" json:"token"`
}

type ConversationMetadata struct {
	Topic     string `bson:"topic,omitempty" json:"topic,omitempty"`
	Sentiment string `bson:"sentiment,omitempty" json:"sentiment,omitempty"`
	Language  string `bson:"language,omitempty" json:"language,omitempty"`
}

type Conversation struct {
	ID        primitive.ObjectID   `bson:"_id,omitempty" json:"id,omitempty"`
	UserID    primitive.ObjectID   `bson:"user_id" json:"user_id"`
	StartedAt time.Time            `bson:"started_at" json:"started_at"`
	UpdatedAt time.Time            `bson:"updated_at" json:"updated_at"`
	Messages  []Message            `bson:"messages" json:"messages"`
	Topic     string               `bson:"topic,omitempty" json:"topic,omitempty"`
	Metadata  ConversationMetadata `bson:"metadata,omitempty" json:"metadata,omitempty"`
}
type ConversationSummary struct {
	ID        primitive.ObjectID `bson:"_id" json:"id"`
	Topic     string             `bson:"topic" json:"topic"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

func NewConversation(userID primitive.ObjectID, content string) *Conversation {
	content = utils.CleanString(content)
	return &Conversation{
		UserID:    userID,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages: []Message{{
			Sender:    "user",
			Content:   content,
			Timestamp: time.Now(),
			Token:     utils.CountToken(content),
		}},
	}
}
// This function generates a response from the user's message using the model API and sends it to the user via websocket.
func GenerateResponseAndWebsocket(userID, content string) string {
	var completeResponse strings.Builder
	isFirstToken := true
	for token := range chatbotapi.GetStreamingResponseFromModelAPI(content) {
		fmt.Print(token) // Print each token for debugging/viewing
		// HANDLE WEBSOCKET HERE
		// Add space before token, except for the first token
		if !isFirstToken {
			ws.BroadcastToken(userID, token + " ")
			completeResponse.WriteString(" ")
		} else {
			isFirstToken = false
			ws.BroadcastToken(userID, token)
		}
		completeResponse.WriteString(token)
	}
	return completeResponse.String()
}
//This function first creates a new conversation with the user's message, then generates a response using the model API and sends it to the user via websocket. Finally, it saves the conversation to the database.
func AskNewConversation(userID primitive.ObjectID, content string, client *mongo.Client) (primitive.ObjectID, error) {
	conversation := NewConversation(userID, content)
	finalResponse := GenerateResponseAndWebsocket(userID.Hex(), conversation.Messages[0].Content)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	collection := client.Database("chatbot-server").Collection("conversation")
	conversation.AddMessage("bot", finalResponse)
	result, err := collection.InsertOne(ctx, conversation)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return result.InsertedID.(primitive.ObjectID), nil
}

func (c *Conversation) AddMessage(sender, content string) {
	c.Messages = append(c.Messages, Message{
		Sender:    sender,
		Content:   content,
		Timestamp: time.Now(),
		Token:     utils.CountToken(content),
	})
	c.UpdatedAt = time.Now()
}
func (c *Conversation) RemoveMessage(index int) {
	c.Messages = append(c.Messages[:index], c.Messages[index+1:]...)
}
// This function generates a response from the whole conversation using the model API and sends it to the user via websocket. It then saves the question and answer to the database.
func AskInConversation(conversationID primitive.ObjectID, content string, client *mongo.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	content = utils.CleanString(content)
	finalResponse := GenerateResponseAndWebsocket("", content)
	filter := bson.M{"_id": conversationID}
	newMessages := []Message{
		{
			Sender:    "user",
			Content:   content,
			Timestamp: time.Now(),
			Token:     utils.CountToken(content),
		},
		{
			Sender:    "bot",
			Content:   finalResponse,
			Timestamp: time.Now(),
			Token:     utils.CountToken(finalResponse),
		},
	}
	update := bson.M{
		"$push": bson.M{
			"messages": bson.M{
				"$each": newMessages,
			},
		},
		"$set": bson.M{"updated_at": time.Now()},
	}
	collection := client.Database("chatbot-server").Collection("conversation")
	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}
// This function retrieves all conversations of a user from the database, it will be sorted based on the last updated_at time. (Warning: pagination is not implemented in this function)
func GetUserConversations(userID primitive.ObjectID, client *mongo.Client) (*[]ConversationSummary, error) {
	filter := bson.M{"user_id": userID}
	projection := bson.M{"_id": 1, "topic": 1}

	findOptions := options.Find().SetProjection(projection).SetSort(bson.M{"updated_at": -1})
	collection := client.Database("chatbot-server").Collection("conversation")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var conversations []ConversationSummary
	if err = cursor.All(ctx, &conversations); err != nil {
		return nil, err
	}

	return &conversations, nil
}
// This function retrieves a single conversation of a user from the database. If the conversation does not exist or does not belong to the user, it returns an error.
func GetOneConversation(conversationID, userID primitive.ObjectID, client *mongo.Client) (*Conversation, error) {
	collection := client.Database("chatbot-server").Collection("conversation")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{
		"_id":     conversationID,
		"user_id": userID,
	}

	var conversation Conversation
	err := collection.FindOne(ctx, filter).Decode(&conversation)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("conversation not found or does not belong to the user")
		}
		return nil, err
	}

	return &conversation, nil
}
