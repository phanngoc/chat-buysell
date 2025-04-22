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

// MatchingResult represents a matching post result with score
type MatchingResult struct {
	Post  Post    `json:"post"`
	User  User    `json:"user"`
	Score float64 `json:"score"`
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

// SearchMatchingPosts searches for matching posts based on post information
// If postType is "mua", it will search for "ban" posts and vice versa
func SearchMatchingPosts(ctx context.Context, postInfo *PostInfo, page, pageSize int) ([]MatchingResult, int, error) {
	if ElasticClient == nil {
		return nil, 0, fmt.Errorf("Elasticsearch client not initialized")
	}
	
	// Determine opposite post type for matching
	oppositeType := "mua"
	if postInfo.Type == "mua" {
		oppositeType = "ban"
	}
	
	// Calculate from for pagination
	from := (page - 1) * pageSize
	
	// Build a query that matches on multiple fields with different weights
	// We'll use should clauses to boost matching on important fields
	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							"post_type": oppositeType,
						},
					},
				},
				"should": []map[string]interface{}{},
			},
		},
		"from": from,
		"size": pageSize,
	}
	
	// Add should clauses for boosting relevant matches
	shouldClauses := []map[string]interface{}{}
	
	// Match on category with high boost
	if postInfo.Category != "" {
		shouldClauses = append(shouldClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"category": map[string]interface{}{
					"value": postInfo.Category,
					"boost": 3.0,
				},
			},
		})
	}
	
	// Match on location
	if postInfo.Location != "" {
		shouldClauses = append(shouldClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"location": map[string]interface{}{
					"value": postInfo.Location,
					"boost": 2.0,
				},
			},
		})
	}
	
	// Match on condition
	if postInfo.Condition != "" {
		shouldClauses = append(shouldClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"condition": map[string]interface{}{
					"value": postInfo.Condition,
					"boost": 1.5,
				},
			},
		})
	}
	
	// Match on price range (if specified)
	if postInfo.Price > 0 {
		// For buying posts looking for selling posts, we want price <= the max the buyer is willing to pay
		// For selling posts looking for buying posts, we want price >= the min the seller is asking
		var priceQuery map[string]interface{}
		
		if postInfo.Type == "mua" {
			// Buyer looking for sellers, want prices less than or equal
			priceQuery = map[string]interface{}{
				"range": map[string]interface{}{
					"price": map[string]interface{}{
						"lte": postInfo.Price,
						// Allow some flexibility in price (about 10% higher)
						"gte": int(float64(postInfo.Price) * 0.5),
					},
				},
			}
		} else {
			// Seller looking for buyers, want prices greater than or equal
			priceQuery = map[string]interface{}{
				"range": map[string]interface{}{
					"price": map[string]interface{}{
						"gte": postInfo.Price,
						// Allow some flexibility (about 50% higher)
						"lte": int(float64(postInfo.Price) * 1.5),
					},
				},
			}
		}
		
		shouldClauses = append(shouldClauses, priceQuery)
	}
	
	// Match on keywords
	if len(postInfo.Keywords) > 0 {
		keywordsQuery := map[string]interface{}{
			"terms": map[string]interface{}{
				"keywords": map[string]interface{}{
					"terms": postInfo.Keywords,
					"boost": 2.0,
				},
			},
		}
		shouldClauses = append(shouldClauses, keywordsQuery)
		
		// Also search in content field for similar terms
		for _, keyword := range postInfo.Keywords {
			contentQuery := map[string]interface{}{
				"match": map[string]interface{}{
					"content": map[string]interface{}{
						"query": keyword,
						"boost": 1.0,
					},
				},
			}
			shouldClauses = append(shouldClauses, contentQuery)
		}
	}
	
	// Add should clauses to query
	searchQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})["should"] = shouldClauses
	
	// Must have at least one should clause match
	if len(shouldClauses) > 0 {
		searchQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})["minimum_should_match"] = 1
	}
	
	// Convert to JSON
	data, err := json.Marshal(searchQuery)
	if err != nil {
		return nil, 0, fmt.Errorf("error marshaling search query: %w", err)
	}
	
	// Perform the search
	res, err := ElasticClient.Search(
		ElasticClient.Search.WithContext(ctx),
		ElasticClient.Search.WithIndex("chat_messages"),
		ElasticClient.Search.WithBody(bytes.NewReader(data)),
		ElasticClient.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("error searching: %w", err)
	}
	defer res.Body.Close()
	
	if res.IsError() {
		return nil, 0, fmt.Errorf("error searching: %s", res.String())
	}
	
	// Parse the response
	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("error parsing search response: %w", err)
	}
	
	// Extract total hits
	total := int(result["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	
	// Extract hits
	hits := result["hits"].(map[string]interface{})["hits"].([]interface{})
	matchResults := make([]MatchingResult, 0, len(hits))
	
	// Process each hit
	for _, hit := range hits {
		source := hit.(map[string]interface{})["_source"].(map[string]interface{})
		score := hit.(map[string]interface{})["_score"].(float64)
		
		var chatMsg ChatMessageIndex
		// Convert to ChatMessageIndex
		data, err := json.Marshal(source)
		if err != nil {
			return nil, 0, fmt.Errorf("error marshaling hit source: %w", err)
		}
		
		if err := json.Unmarshal(data, &chatMsg); err != nil {
			return nil, 0, fmt.Errorf("error unmarshaling hit source: %w", err)
		}
		
		// Skip if there's no post ID
		if chatMsg.PostID == "" {
			continue
		}
		
		// Create a partial result with the score
		matchResult := MatchingResult{
			Score: score,
		}
		
		// We'll fill in Post and User details later
		
		matchResults = append(matchResults, matchResult)
	}
	
	return matchResults, total, nil
}

// GetMatchingPosts finds matching posts for the given post content
// It first classifies the content, then searches for matching posts
func GetMatchingPosts(ctx context.Context, db *mongo.Database, content string, page, pageSize int) ([]MatchingResult, int, error) {
	// First, classify the post content
	postInfo, err := ClassifyPost(ctx, content)
	if err != nil {
		return nil, 0, fmt.Errorf("error classifying post: %w", err)
	}
	
	// Now find matching posts in Elasticsearch
	matchResults, total, err := SearchMatchingPosts(ctx, postInfo, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("error searching matching posts: %w", err)
	}
	
	// Get details for each match from MongoDB
	for i := range matchResults {
		if i < len(matchResults) {
			// Get post details
			postID, err := primitive.ObjectIDFromHex(matchResults[i].Post.ID.Hex())
			if err == nil {
				var post Post
				err = db.Collection("posts").FindOne(ctx, bson.M{"_id": postID}).Decode(&post)
				if err == nil {
					matchResults[i].Post = post
				}
			}
			
			// Get user details
			if !matchResults[i].Post.UserID.IsZero() {
				var user User
				err = db.Collection("users").FindOne(ctx, bson.M{"_id": matchResults[i].Post.UserID}).Decode(&user)
				if err == nil {
					matchResults[i].User = user
				}
			}
		}
	}
	
	return matchResults, total, nil
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
