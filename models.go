package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// User struct
// type: buyer | seller
// uid: Facebook ID hoặc hệ thống
// Thêm các trường mở rộng nếu cần

type User struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UID         string             `bson:"uid" json:"uid"`
	Username    string             `bson:"username" json:"username"`
	Avatar      string             `bson:"avatar" json:"avatar"`
	Type        string             `bson:"type" json:"type"`
	Email       string             `bson:"email" json:"email"`
	AccessToken string             `bson:"accessToken" json:"accessToken"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
}

// Post struct
// type: 'mua' | 'ban'
type Post struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Type      string             `bson:"type" json:"type"`
	Content   string             `bson:"content" json:"content"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UserID    primitive.ObjectID `bson:"userId" json:"userId"`
	Category  string             `json:"category"`
	Location  string             `json:"location"`
	Price     int                `json:"price"`
	Condition string             `json:"condition"`
	Keywords  []string           `json:"keywords"`
}

// PostInfo struct for NLP classification results
type PostInfo struct {
	Type      string   `json:"type"`
	Category  string   `json:"category"`
	Location  string   `json:"location"`
	Price     int      `json:"price"`
	Condition string   `json:"condition"`
	Keywords  []string `json:"keywords"`
}

// Message struct
type Message struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RoomID    primitive.ObjectID `bson:"roomId" json:"roomId"`
	SenderID  primitive.ObjectID `bson:"senderId" json:"senderId"`
	Content   string             `bson:"content" json:"content"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}

// ChatRoom struct
type ChatRoom struct {
	ID        primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	BuyerID   primitive.ObjectID   `bson:"buyerId" json:"buyerId"`
	SellerID  primitive.ObjectID   `bson:"sellerId" json:"sellerId"`
	PostID    primitive.ObjectID   `bson:"postId" json:"postId"`
	Messages  []primitive.ObjectID `bson:"messages" json:"messages"`
	CreatedAt time.Time            `bson:"createdAt" json:"createdAt"`
}

// ChatMessageIndex represents the structure for chat messages in Elasticsearch
type ChatMessageIndex struct {
	ID          string    `json:"id"`
	RoomID      string    `json:"room_id"`
	SenderID    string    `json:"sender_id"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	PostType    string    `json:"post_type,omitempty"`    // "mua" or "ban"
	Category    string    `json:"category,omitempty"`
	Location    string    `json:"location,omitempty"`
	Price       int       `json:"price,omitempty"`
	Condition   string    `json:"condition,omitempty"`
	Keywords    []string  `json:"keywords,omitempty"`
	BuyerID     string    `json:"buyer_id,omitempty"`
	SellerID    string    `json:"seller_id,omitempty"`
	PostID      string    `json:"post_id,omitempty"`
	Classified  bool      `json:"classified"`          // Whether this message has been classified
	MessageType string    `json:"message_type"`        // "question", "negotiation", "agreement", etc.
}

// ElasticClient holds the Elasticsearch client
var ElasticClient *elasticsearch.Client

// InitElasticsearch initializes the Elasticsearch connection
func InitElasticsearch(url string) error {
	cfg := elasticsearch.Config{
		Addresses: []string{url},
	}
	
	var err error
	ElasticClient, err = elasticsearch.NewClient(cfg)
	if (err != nil) {
		return fmt.Errorf("error creating Elasticsearch client: %w", err)
	}
	
	// Check the connection
	res, err := ElasticClient.Info()
	if (err != nil) {
		return fmt.Errorf("error getting Elasticsearch info: %w", err)
	}
	defer res.Body.Close()
	
	// Ensure the index exists
	if (err := createChatMessagesIndex(); err != nil) {
		return fmt.Errorf("error creating chat messages index: %w", err)
	}
	
	return nil
}

// createChatMessagesIndex creates the chat_messages index if it doesn't exist
func createChatMessagesIndex() error {
	// Define the mapping for chat messages
	mapping := `{
		"settings": {
			"number_of_shards": 1,
			"number_of_replicas": 0,
			"analysis": {
				"analyzer": {
					"vietnamese_analyzer": {
						"type": "custom",
						"tokenizer": "standard",
						"filter": ["lowercase", "asciifolding"]
					}
				}
			}
		},
		"mappings": {
			"properties": {
				"id": { "type": "keyword" },
				"room_id": { "type": "keyword" },
				"sender_id": { "type": "keyword" },
				"content": { 
					"type": "text",
					"analyzer": "vietnamese_analyzer" 
				},
				"created_at": { "type": "date" },
				"post_type": { "type": "keyword" },
				"category": { "type": "keyword" },
				"location": { "type": "keyword" },
				"price": { "type": "integer" },
				"condition": { "type": "keyword" },
				"keywords": { "type": "keyword" },
				"buyer_id": { "type": "keyword" },
				"seller_id": { "type": "keyword" },
				"post_id": { "type": "keyword" },
				"classified": { "type": "boolean" },
				"message_type": { "type": "keyword" }
			}
		}
	}`
	
	req := esapi.IndicesCreateRequest{
		Index: "chat_messages",
		Body:  bytes.NewReader([]byte(mapping)),
	}
	
	res, err := req.Do(context.Background(), ElasticClient)
	if (err != nil) {
		return err
	}
	defer res.Body.Close()
	
	// If the index already exists, that's fine
	if (res.StatusCode == 400) {
		var r map[string]interface{}
		if (err := json.NewDecoder(res.Body).Decode(&r); err != nil) {
			return err
		}
		
		// Check if the error is because the index already exists
		if (r["error"].(map[string]interface{})["type"].(string) == "resource_already_exists_exception") {
			return nil
		}
		
		return fmt.Errorf("error creating index: %v", r["error"])
	}
	
	if (res.IsError()) {
		return fmt.Errorf("error creating index: %s", res.String())
	}
	
	return nil
}

// IndexChatMessage indexes a chat message in Elasticsearch
func IndexChatMessage(ctx context.Context, msg Message, chatRoom *ChatRoom, post *Post) error {
	if (ElasticClient == nil) {
		return fmt.Errorf("Elasticsearch client not initialized")
	}
	
	// Create a document to index
	chatMsg := ChatMessageIndex{
		ID:         msg.ID.Hex(),
		RoomID:     msg.RoomID.Hex(),
		SenderID:   msg.SenderID.Hex(),
		Content:    msg.Content,
		CreatedAt:  msg.CreatedAt,
		Classified: false, // Default to not classified
	}
	
	// Add additional context if ChatRoom is provided
	if (chatRoom != nil) {
		chatMsg.BuyerID = chatRoom.BuyerID.Hex()
		chatMsg.SellerID = chatRoom.SellerID.Hex()
		chatMsg.PostID = chatRoom.PostID.Hex()
	}
	
	// Add post details if available
	if (post != nil) {
		chatMsg.PostType = post.Type
		chatMsg.Category = post.Category
		chatMsg.Location = post.Location
		chatMsg.Price = post.Price
		chatMsg.Condition = post.Condition
		chatMsg.Keywords = post.Keywords
	}
	
	// Convert to JSON
	data, err := json.Marshal(chatMsg)
	if (err != nil) {
		return fmt.Errorf("error marshaling chat message: %w", err)
	}
	
	// Index the document
	req := esapi.IndexRequest{
		Index:      "chat_messages",
		DocumentID: chatMsg.ID,
		Body:       bytes.NewReader(data),
		Refresh:    "true",
	}
	
	res, err := req.Do(ctx, ElasticClient)
	if (err != nil) {
		return fmt.Errorf("error indexing chat message: %w", err)
	}
	defer res.Body.Close()
	
	if (res.IsError()) {
		return fmt.Errorf("error indexing document: %s", res.String())
	}
	
	return nil
}

// SearchChatMessages searches for chat messages in Elasticsearch
func SearchChatMessages(ctx context.Context, query string, from, size int) ([]ChatMessageIndex, int, error) {
	if (ElasticClient == nil) {
		return nil, 0, fmt.Errorf("Elasticsearch client not initialized")
	}
	
	// Build the search request
	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  query,
				"fields": []string{"content", "category", "location", "keywords^2"},
			},
		},
		"sort": []map[string]interface{}{
			{
				"created_at": map[string]interface{}{
					"order": "desc",
				},
			},
		},
		"from": from,
		"size": size,
	}
	
	// Convert to JSON
	data, err := json.Marshal(searchQuery)
	if (err != nil) {
		return nil, 0, fmt.Errorf("error marshaling search query: %w", err)
	}
	
	// Perform the search
	res, err := ElasticClient.Search(
		ElasticClient.Search.WithContext(ctx),
		ElasticClient.Search.WithIndex("chat_messages"),
		ElasticClient.Search.WithBody(bytes.NewReader(data)),
		ElasticClient.Search.WithTrackTotalHits(true),
	)
	if (err != nil) {
		return nil, 0, fmt.Errorf("error searching: %w", err)
	}
	defer res.Body.Close()
	
	if (res.IsError()) {
		return nil, 0, fmt.Errorf("error searching: %s", res.String())
	}
	
	// Parse the response
	var result map[string]interface{}
	if (err := json.NewDecoder(res.Body).Decode(&result); err != nil) {
		return nil, 0, fmt.Errorf("error parsing search response: %w", err)
	}
	
	// Extract total hits
	total := int(result["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	
	// Extract hits
	hits := result["hits"].(map[string]interface{})["hits"].([]interface{})
	messages := make([]ChatMessageIndex, 0, len(hits))
	
	for _, hit := range hits {
		source := hit.(map[string]interface{})["_source"].(map[string]interface{})
		var msg ChatMessageIndex
		
		// Convert source to JSON and then to struct
		data, err := json.Marshal(source)
		if (err != nil) {
			return nil, 0, fmt.Errorf("error marshaling hit source: %w", err)
		}
		
		if (err := json.Unmarshal(data, &msg); err != nil) {
			return nil, 0, fmt.Errorf("error unmarshaling hit source: %w", err)
		}
		
		messages = append(messages, msg)
	}
	
	return messages, total, nil
}

// ClassifyChatMessage classifies a chat message and updates its Elasticsearch document
func ClassifyChatMessage(ctx context.Context, msgID string, messageType string) error {
	if (ElasticClient == nil) {
		return fmt.Errorf("Elasticsearch client not initialized")
	}
	
	// Update document
	updateDoc := map[string]interface{}{
		"doc": map[string]interface{}{
			"classified":   true,
			"message_type": messageType,
		},
	}
	
	data, err := json.Marshal(updateDoc)
	if (err != nil) {
		return fmt.Errorf("error marshaling update doc: %w", err)
	}
	
	req := esapi.UpdateRequest{
		Index:      "chat_messages",
		DocumentID: msgID,
		Body:       bytes.NewReader(data),
		Refresh:    "true",
	}
	
	res, err := req.Do(ctx, ElasticClient)
	if (err != nil) {
		return fmt.Errorf("error updating document: %w", err)
	}
	defer res.Body.Close()
	
	if (res.IsError()) {
		return fmt.Errorf("error updating document: %s", res.String())
	}
	
	return nil
}

// Example: insert user
func InsertUser(ctx context.Context, db *mongo.Database, user User) (*mongo.InsertOneResult, error) {
	user.CreatedAt = time.Now()
	return db.Collection("users").InsertOne(ctx, user)
}

// Example: insert post
func InsertPost(ctx context.Context, db *mongo.Database, post Post) (*mongo.InsertOneResult, error) {
	post.CreatedAt = time.Now()
	return db.Collection("posts").InsertOne(ctx, post)
}

// Example: insert message
func InsertMessage(ctx context.Context, db *mongo.Database, msg Message) (*mongo.InsertOneResult, error) {
	msg.CreatedAt = time.Now()
	return db.Collection("messages").InsertOne(ctx, msg)
}

// Example: insert chatroom
func InsertChatRoom(ctx context.Context, db *mongo.Database, room ChatRoom) (*mongo.InsertOneResult, error) {
	room.CreatedAt = time.Now()
	return db.Collection("chatrooms").InsertOne(ctx, room)
}
