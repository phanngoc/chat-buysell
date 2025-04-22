package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
)

var (
	facebookOauthConfig = &oauth2.Config{
		RedirectURL:  "http://localhost:8080/auth/facebook/callback",
		ClientID:     os.Getenv("FACEBOOK_CLIENT_ID"),
		ClientSecret: os.Getenv("FACEBOOK_CLIENT_SECRET"),
		Scopes:       []string{"email", "public_profile"},
		Endpoint:     facebook.Endpoint,
	}
	mongoClient        *mongo.Client
	userCollection     *mongo.Collection
	postCollection     *mongo.Collection
	messageCollection  *mongo.Collection
	chatroomCollection *mongo.Collection
	mongoDB            *mongo.Database
)

var ErrNoAPIKey = errors.New("OPENAI_API_KEY not set")

func main() {
	// Example usage of models package
	var post Post
	fmt.Println(post)

	// MongoDB connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var err error

	// Connect to MongoDB
	mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatalf("MongoDB connect error: %v", err)
	}

	// Initialize MongoDB collections
	mongoDB = mongoClient.Database("chatbuysell")
	userCollection = mongoDB.Collection("users")
	postCollection = mongoDB.Collection("posts")
	messageCollection = mongoDB.Collection("messages")
	chatroomCollection = mongoDB.Collection("chatrooms")

	// Initialize Elasticsearch
	if err := InitElasticsearch("http://localhost:9200"); err != nil {
		log.Printf("Warning: Failed to initialize Elasticsearch: %v", err)
		// Continue anyway, as Elasticsearch is optional
	} else {
		log.Println("Elasticsearch initialized successfully")
	}

	r := gin.Default()

	// Auth routes
	r.GET("/auth/facebook", handleFacebookLogin)
	r.GET("/auth/facebook/callback", handleFacebookCallback)

	// NLP routes
	r.POST("/nlp/classify", handleNLPClassify)

	// Chat routes
	r.POST("/chat/message", handleCreateMessage)
	r.GET("/chat/room/:id", handleGetChatRoom)

	// Search routes
	r.GET("/search/chat", handleSearchChat)
	r.POST("/chat/classify", handleClassifyMessage)

	r.Run(":8080")
}

func handleFacebookLogin(c *gin.Context) {
	url := facebookOauthConfig.AuthCodeURL("state", oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func handleFacebookCallback(c *gin.Context) {
	state := c.Query("state")
	if state != "state" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid OAuth state"})
		return
	}
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Code not found"})
		return
	}
	token, err := facebookOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token exchange failed"})
		return
	}
	client := facebookOauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://graph.facebook.com/me?fields=id,email,picture.type(large)&access_token=" + token.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}
	defer resp.Body.Close()
	var fbUser struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Picture struct {
			Data struct {
				URL string `json:"url"`
			} `json:"data"`
		} `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&fbUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Decode user info failed"})
		return
	}
	user := User{
		UID:         fbUser.ID,
		Email:       fbUser.Email,
		Avatar:      fbUser.Picture.Data.URL,
		AccessToken: token.AccessToken,
	}
	// Upsert user
	filter := bson.M{"uid": user.UID}
	update := bson.M{"$set": user}
	opts := options.Update().SetUpsert(true)
	_, err = userCollection.UpdateOne(context.Background(), filter, update, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB upsert failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func handleNLPClassify(c *gin.Context) {
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "detail": err.Error()})
		return
	}
	ctx := context.Background()
	result, err := ClassifyPost(ctx, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "NLP error", "detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// handleCreateMessage creates a new chat message and indexes it in Elasticsearch
func handleCreateMessage(c *gin.Context) {
	var req struct {
		RoomID   string `json:"roomId"`
		SenderID string `json:"senderId"`
		Content  string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "detail": err.Error()})
		return
	}

	// Parse ObjectIDs
	roomID, err := primitive.ObjectIDFromHex(req.RoomID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	senderID, err := primitive.ObjectIDFromHex(req.SenderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sender ID"})
		return
	}

	// Create message
	msg := Message{
		ID:        primitive.NewObjectID(),
		RoomID:    roomID,
		SenderID:  senderID,
		Content:   req.Content,
		CreatedAt: time.Now(),
	}

	ctx := context.Background()

	// Insert into MongoDB
	result, err := InsertMessage(ctx, mongoDB, msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create message", "detail": err.Error()})
		return
	}

	// Update chat room with message ID
	_, err = chatroomCollection.UpdateOne(
		ctx,
		bson.M{"_id": roomID},
		bson.M{"$push": bson.M{"messages": msg.ID}},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update chat room"})
		return
	}

	// Index in Elasticsearch (if enabled)
	if ElasticClient != nil {
		// Get chat room and post details for context
		var chatRoom ChatRoom
		err := chatroomCollection.FindOne(ctx, bson.M{"_id": roomID}).Decode(&chatRoom)
		if err != nil && err != mongo.ErrNoDocuments {
			log.Printf("Warning: Error getting chat room details: %v", err)
		}

		var post Post
		if chatRoom.PostID != primitive.NilObjectID {
			err = postCollection.FindOne(ctx, bson.M{"_id": chatRoom.PostID}).Decode(&post)
			if err != nil && err != mongo.ErrNoDocuments {
				log.Printf("Warning: Error getting post details: %v", err)
			}
		}

		// Index the message with context
		if err := IndexChatMessage(ctx, msg, &chatRoom, &post); err != nil {
			log.Printf("Warning: Error indexing chat message in Elasticsearch: %v", err)
			// Continue anyway, as Elasticsearch indexing should not block the API response
		}
	}

	c.JSON(http.StatusOK, gin.H{"messageId": msg.ID, "insertResult": result})
}

// handleGetChatRoom retrieves a chat room and its messages
func handleGetChatRoom(c *gin.Context) {
	roomID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	ctx := context.Background()

	var chatRoom ChatRoom
	if err := chatroomCollection.FindOne(ctx, bson.M{"_id": roomID}).Decode(&chatRoom); err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chat room not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	// Get messages in the chat room
	cursor, err := messageCollection.Find(
		ctx,
		bson.M{"roomId": roomID},
		options.Find().SetSort(bson.M{"createdAt": 1}),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve messages"})
		return
	}
	defer cursor.Close(ctx)

	var messages []Message
	if err := cursor.All(ctx, &messages); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse messages"})
		return
	}

	// Get buyer and seller info
	var buyer User
	var seller User

	buyerErr := userCollection.FindOne(ctx, bson.M{"_id": chatRoom.BuyerID}).Decode(&buyer)
	sellerErr := userCollection.FindOne(ctx, bson.M{"_id": chatRoom.SellerID}).Decode(&seller)

	// Get post info
	var post Post
	postErr := postCollection.FindOne(ctx, bson.M{"_id": chatRoom.PostID}).Decode(&post)

	response := gin.H{
		"chatRoom": chatRoom,
		"messages": messages,
	}

	if buyerErr == nil {
		response["buyer"] = buyer
	}

	if sellerErr == nil {
		response["seller"] = seller
	}

	if postErr == nil {
		response["post"] = post
	}

	c.JSON(http.StatusOK, response)
}

// handleSearchChat searches chat messages in Elasticsearch
func handleSearchChat(c *gin.Context) {
	if ElasticClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Search service not available"})
		return
	}

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	from := (page - 1) * pageSize

	ctx := c.Request.Context()
	messages, total, err := SearchChatMessages(ctx, query, from, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// handleClassifyMessage classifies a chat message and updates its classification in Elasticsearch
func handleClassifyMessage(c *gin.Context) {
	if ElasticClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Classification service not available"})
		return
	}

	var req struct {
		MessageID   string `json:"messageId"`
		MessageType string `json:"messageType"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "detail": err.Error()})
		return
	}

	if req.MessageID == "" || req.MessageType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message ID and type are required"})
		return
	}

	// Valid message types
	validTypes := map[string]bool{
		"question":    true,
		"negotiation": true,
		"agreement":   true,
		"inquiry":     true,
		"other":       true,
	}

	if !validTypes[req.MessageType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message type"})
		return
	}

	ctx := c.Request.Context()
	if err := ClassifyChatMessage(ctx, req.MessageID, req.MessageType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Classification failed", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ClassifyPost sử dụng OpenAI để phân tích nội dung tin đăng
func ClassifyPost(ctx context.Context, content string) (*PostInfo, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, ErrNoAPIKey
	}

	client := openai.NewClient(apiKey)

	systemContent := "Bạn là một AI phân loại tin đăng mua bán. Hãy trích xuất các trường dưới dạng JSON: type (mua|bán), category, location, price (số nguyên, nếu không có thì để 0), condition, keywords (mảng 3-5 từ khóa). Nếu không có trường nào thì để rỗng hoặc 0."
	userContent := "Nội dung tin đăng: " + content + "\nHãy trả về kết quả JSON."

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemContent,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userContent,
			},
		},
		Temperature: 0.3,
	})
	if err != nil {
		return nil, err
	}

	var info PostInfo
	jsonStr := resp.Choices[0].Message.Content
	jsonStr = strings.TrimSpace(jsonStr)
	err = json.Unmarshal([]byte(jsonStr), &info)
	if err != nil {
		return nil, err
	}

	return &info, nil
}
