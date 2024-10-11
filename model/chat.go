package model

import (
	"context"
	"server/chatbotAPI"
	"server/utils"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Message struct {
	Sender    string    `bson:"sender" json:"sender"`
	Content   string    `bson:"content" json:"content"`
	Timestamp time.Time `bson:"timestamp" json:"timestamp"`
	Token int `bson:"token" json:"token"`
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
	Metadata  ConversationMetadata `bson:"metadata,omitempty" json:"metadata,omitempty"`
}
func NewConversation(userID primitive.ObjectID, content string) *Conversation {
	content = utils.CleanString(content)
	return &Conversation{
		UserID:    userID,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages:  []Message{{
			Sender:    "user",
			Content: content,
			Timestamp: time.Now(),
			Token:utils.CountToken(content),
		}},
	}
}
func CreateConversation(conversation *Conversation,client *mongo.Client) (primitive.ObjectID, error) {
	chatbotapi.GetStreamingResponseFromModelAPI(conversation.Messages[0].Content)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db := client.Database("chatbot-server")
	collection := db.Collection("conversation")
	result, err := collection.InsertOne(ctx, conversation)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return result.InsertedID.(primitive.ObjectID), nil
}
func (c *Conversation) AddMessage(sender, content string) {
	c.Messages = append(c.Messages, Message{
		Sender:    sender,
		Content: content,
		Timestamp: time.Now(),
	})
	c.UpdatedAt = time.Now()
}
func (c *Conversation) RemoveMessage(index int) {
	c.Messages = append(c.Messages[:index], c.Messages[index+1:]...)
}