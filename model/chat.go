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
	Cid       string    `bson:"cid,omitempty" json:"cid,omitempty"`
	Timestamp time.Time `bson:"timestamp" json:"timestamp"`
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
	Mode      string               `bson:"mode,omitempty" json:"mode,omitempty"`
}
type ConversationSummary struct {
	ID        primitive.ObjectID `bson:"_id" json:"id"`
	Topic     string             `bson:"topic" json:"topic"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
	Mode      string             `bson:"mode" json:"mode"`
}

func NewConversation(userID primitive.ObjectID, content string, cid string) (*Conversation, error) {
	content = utils.CleanString(content)
	return &Conversation{
		UserID:    userID,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
		Topic:     "",
		Messages: []Message{{
			Sender:    "user",
			Content:   content,
			Timestamp: time.Now(),
			Cid:       cid,
		}},
	}, nil
}

// This function generates a response from the user's message using the model API and sends it to the user via websocket.
func GenerateResponseAndWebsocket(userID, content, id, mode string, isFirst bool, cid string) (string, string, error) {
	var completeResponse strings.Builder
	prefix := "Chủ đề-123: "
	if userID == "" {
		return "", "", errors.New("userID is empty")
	}
	if id == "" {
		return "", "", errors.New("chatID is empty")
	}
	clientID := userID + ":" + id
	if client, exists := ws.Clients[clientID]; exists {
		if client.IsSending {
			return "", "", errors.New("client is sending")
		}
		client.Mu.Lock()
		client.IsSending = true
		client.Mu.Unlock()
	}

	for token := range chatbotapi.GetStreamingResponseFromModelAPI(content, mode, id, isFirst, cid) {
		// Print each token for debugging/viewing
		// HANDLE WEBSOCKET HERE
		ws.BroadcastToken(userID, id, token)
		if strings.HasPrefix(token, prefix) {
			topic := token[len(prefix):]
			ws.BroadcastToken(userID, id, "end of response")
			if strings.HasSuffix(topic, "\n") {
				topic = topic[:len(topic)-1]
			}
			return completeResponse.String(), topic, nil
		}
		completeResponse.WriteString(token)

	}
	ws.BroadcastToken(userID, id, "end of response")
	if client, exists := ws.Clients[clientID]; exists {
		client.Mu.Lock()
		client.IsSending = false
		client.Mu.Unlock()
	}
	return completeResponse.String(), "", nil
}
func CheckConversationUser(userID, conversationID primitive.ObjectID, client *mongo.Client) error {
	collection := client.Database("chatbot-server").Collection("conversation")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	filter := bson.M{
		"_id":     conversationID,
		"user_id": userID,
	}

	var conversation Conversation
	err := collection.FindOne(ctx, filter).Decode(&conversation)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New("conversation not found or does not belong to the user")
		}
		return err
	}

	return nil
}

// This function first creates a new conversation with the user's message, then generates a response using the model API and sends it to the user via websocket. Finally, it saves the conversation to the database.
func AskNewConversation(userID primitive.ObjectID, content string, client *mongo.Client, mode string, cid string) (primitive.ObjectID, error) {
	if content == "" {
		return primitive.NilObjectID, errors.New("content is empty")
	}
	conversation, err := NewConversation(userID, content, cid)
	if err != nil {
		return primitive.NilObjectID, err
	}
	conversation.Mode = mode
	collection := client.Database("chatbot-server").Collection("conversation")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := collection.InsertOne(ctx, conversation)
	if err != nil {
		return primitive.NilObjectID, errors.New("failed to create conversation")
	}

	go func() {
		finalResponse, topic, err := GenerateResponseAndWebsocket(userID.Hex(), conversation.Messages[0].Content, result.InsertedID.(primitive.ObjectID).Hex(), mode, true, cid)
		if err != nil {
			fmt.Println(err)
			return
		}
		filter := bson.M{"_id": result.InsertedID.(primitive.ObjectID)}
		newMessage := Message{
			Sender:    "bot",
			Content:   finalResponse,
			Timestamp: time.Now(),
		}
		update := bson.M{
			"$push": bson.M{"messages": newMessage},
			"$set":  bson.M{"updated_at": time.Now(), "topic": topic},
		}

		if _, err := collection.UpdateOne(context.TODO(), filter, update); err != nil {
			fmt.Println(err)
		}
	}()

	return result.InsertedID.(primitive.ObjectID), nil
}
func (c *Conversation) AddMessage(sender, content string) {
	c.Messages = append(c.Messages, Message{
		Sender:    sender,
		Content:   content,
		Timestamp: time.Now(),
	})
	c.UpdatedAt = time.Now()
}
func (c *Conversation) RemoveMessage(index int) {
	c.Messages = append(c.Messages[:index], c.Messages[index+1:]...)
}

// This function generates a response from the whole conversation using the model API and sends it to the user via websocket. It then saves the question and answer to the database.
func AskInConversation(conversationID primitive.ObjectID, content string, client *mongo.Client, cid string) error {
	if content == "" {
		return errors.New("content is empty")
	}
	content = utils.CleanString(content)

	collection1 := client.Database("chatbot-server").Collection("conversation")
	// Create a filter for the _id
	var result struct {
		UserID primitive.ObjectID `bson:"user_id"`
		Mode   string             `bson:"mode"`
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err1 := collection1.FindOne(
		ctx,
		bson.M{"_id": conversationID},
	).Decode(&result)
	if err1 != nil {
		return err1
	}
	go func() {
		if finalResponse, _, err := GenerateResponseAndWebsocket(result.UserID.Hex(), content, conversationID.Hex(), result.Mode, false, cid); err != nil {
			fmt.Println(err)
			return
		} else {
			filter := bson.M{"_id": conversationID}
			newMessages := []Message{
				{
					Sender:    "user",
					Content:   content,
					Timestamp: time.Now(),
					Cid:       cid,
				},
				{
					Sender:    "bot",
					Content:   finalResponse,
					Timestamp: time.Now(),
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
			_, err := collection.UpdateOne(context.TODO(), filter, update)
			if err != nil {
				fmt.Println(err)
			}
		}
	}()
	return nil
}

// This function retrieves all conversations of a user from the database, it will be sorted based on the last updated_at time. (Warning: pagination is not implemented in this function)
func GetUserConversations(userID primitive.ObjectID, client *mongo.Client) (*[]ConversationSummary, error) {
	filter := bson.M{"user_id": userID}
	projection := bson.M{
		"_id":        1,
		"topic":      1,
		"updated_at": 1,
		"mode":       1,
	}

	findOptions := options.Find().SetProjection(projection).SetSort(bson.M{"updated_at": -1})
	collection := client.Database("chatbot-server").Collection("conversation")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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
func GetUserConversationsPage(userID primitive.ObjectID, client *mongo.Client, page int64) (*[]ConversationSummary, error) {
	filter := bson.M{"user_id": userID}
	projection := bson.M{
		"_id":        1,
		"topic":      1,
		"updated_at": 1,
		"mode":       1,
	}
	skip := (page - 1) * 8
	findOptions := options.Find().SetProjection(projection).SetSort(bson.M{"updated_at": -1}).SetLimit(8).SetSkip(skip)
	collection := client.Database("chatbot-server").Collection("conversation")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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
