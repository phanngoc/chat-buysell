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

	// Matching routes
	r.POST("/matching/find", handleFindMatches)
	r.POST("/post/create", handleCreatePost)
	r.GET("/post/type/:type", handleGetPostsByType)

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

// handleFindMatches finds potential matches based on post content
func handleFindMatches(c *gin.Context) {
	var req struct {
		Content  string `json:"content" binding:"required"`
		Page     int    `json:"page"`
		PageSize int    `json:"pageSize"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "detail": err.Error()})
		return
	}

	// Default pagination values
	if req.Page < 1 {
		req.Page = 1
	}

	if req.PageSize < 1 || req.PageSize > 50 {
		req.PageSize = 10
	}

	ctx := context.Background()

	// First classify the content to extract structured information
	postInfo, err := ClassifyPost(ctx, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to classify post content", "detail": err.Error()})
		return
	}

	// Use the classified information to find matching posts
	matchResults, total, err := GetMatchingPosts(ctx, mongoDB, req.Content, req.Page, req.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find matches", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"matches":  matchResults,
		"total":    total,
		"page":     req.Page,
		"pageSize": req.PageSize,
		"postInfo": postInfo, // Include the classification results
	})
}

// handleCreatePost creates a new post with NLP classification
func handleCreatePost(c *gin.Context) {
	var req struct {
		UserID  string `json:"userId" binding:"required"`
		Content string `json:"content" binding:"required"`
		Type    string `json:"type" binding:"required,oneof=mua ban"` // Explicit type override
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "detail": err.Error()})
		return
	}

	ctx := context.Background()

	// Parse user ID
	userID, err := primitive.ObjectIDFromHex(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Verify user exists
	var user User
	err = userCollection.FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	// Classify post content using NLP
	postInfo, err := ClassifyPost(ctx, req.Content)
	if err != nil {
		log.Printf("Warning: Failed to classify post: %v", err)
		// Continue anyway, using user-provided type
		postInfo = &PostInfo{
			Type:     req.Type,
			Category: "", // Empty fields will be filled by frontend
		}
	} else {
		// Override the NLP classification type with user's explicit choice if provided
		postInfo.Type = req.Type
	}

	// Create and save the post
	post := Post{
		ID:        primitive.NewObjectID(),
		Type:      postInfo.Type,
		Content:   req.Content,
		UserID:    userID,
		CreatedAt: time.Now(),
		Category:  postInfo.Category,
		Location:  postInfo.Location,
		Price:     postInfo.Price,
		Condition: postInfo.Condition,
		Keywords:  postInfo.Keywords,
	}

	result, err := InsertPost(ctx, mongoDB, post)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create post", "detail": err.Error()})
		return
	}

	// Index the post in Elasticsearch for matching
	if ElasticClient != nil {
		// Create a mock message to index the post content
		msg := Message{
			ID:        primitive.NewObjectID(),
			Content:   req.Content,
			CreatedAt: time.Now(),
		}

		// Index in Elasticsearch
		if err := IndexChatMessage(ctx, msg, nil, &post); err != nil {
			log.Printf("Warning: Error indexing post in Elasticsearch: %v", err)
			// Continue anyway, as Elasticsearch indexing should not block the API response
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"post":       post,
		"insertedID": result.InsertedID,
		"postInfo":   postInfo,
	})
}

// handleGetPostsByType retrieves posts by type (mua/ban)
func handleGetPostsByType(c *gin.Context) {
	postType := c.Param("type")
	if postType != "mua" && postType != "ban" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post type. Must be 'mua' or 'ban'"})
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

	// Filter options
	category := c.Query("category")
	location := c.Query("location")
	minPrice, _ := strconv.Atoi(c.DefaultQuery("minPrice", "0"))
	maxPrice, _ := strconv.Atoi(c.DefaultQuery("maxPrice", "0"))

	// Build filter
	filter := bson.M{"type": postType}

	if category != "" {
		filter["category"] = category
	}

	if location != "" {
		filter["location"] = location
	}

	if minPrice > 0 || maxPrice > 0 {
		priceFilter := bson.M{}
		if minPrice > 0 {
			priceFilter["$gte"] = minPrice
		}
		if maxPrice > 0 {
			priceFilter["$lte"] = maxPrice
		}
		filter["price"] = priceFilter
	}

	// Options for sorting and pagination
	findOptions := options.Find().
		SetSort(bson.M{"createdAt": -1}).
		SetSkip(int64((page - 1) * pageSize)).
		SetLimit(int64(pageSize))

	// Execute query
	ctx := context.Background()
	cursor, err := postCollection.Find(ctx, filter, findOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "detail": err.Error()})
		return
	}
	defer cursor.Close(ctx)

	// Get results
	var posts []Post
	if err := cursor.All(ctx, &posts); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing results", "detail": err.Error()})
		return
	}

	// Get total count for pagination
	total, err := postCollection.CountDocuments(ctx, filter)
	if err != nil {
		log.Printf("Warning: Error counting posts: %v", err)
		total = 0
	}

	// For each post, get user info
	type PostWithUser struct {
		Post Post `json:"post"`
		User User `json:"user,omitempty"`
	}

	result := make([]PostWithUser, 0, len(posts))
	for _, post := range posts {
		item := PostWithUser{
			Post: post,
		}

		// Get user
		var user User
		err := userCollection.FindOne(ctx, bson.M{"_id": post.UserID}).Decode(&user)
		if err == nil {
			item.User = user
		}

		result = append(result, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"posts":    result,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
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
