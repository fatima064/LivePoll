package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var (
	db  *mongo.Database
	rdb *redis.Client
	ctx = context.Background()
)

type User struct {
	ID       bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Email    string        `bson:"email" json:"email"`
	Password string        `bson:"password" json:"-"`
}

type PollOption struct {
	ID   string `bson:"id" json:"id"`
	Text string `bson:"text" json:"text"`
}

type Poll struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Question  string        `bson:"question" json:"question"`
	Options   []PollOption  `bson:"options" json:"options"`
	CreatedBy string        `bson:"createdBy" json:"createdBy"`
	CreatedAt time.Time     `bson:"createdAt" json:"createdAt"`
}

type Claims struct {
	UserID string `json:"userId"`
	jwt.RegisteredClaims
}

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: .env file not found")
	}

	// ---------- MongoDB ----------
	mongoURI := os.Getenv("MONGO_URI")

	mongoClient, err := mongo.Connect(
		options.Client().ApplyURI(mongoURI),
	)
	if err != nil {
		panic(err)
	}

	if err := mongoClient.Ping(ctx, nil); err != nil {
		panic(err)
	}

	db = mongoClient.Database("livepoll")

	// ---------- Redis ----------
	redisURL := os.Getenv("REDIS_URL")

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		panic(err)
	}

	rdb = redis.NewClient(opt)

	if err := rdb.Ping(ctx).Err(); err != nil {
		panic(err)
	}

	// ---------- Gin ----------
	r := gin.Default()

	// CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set(
			"Access-Control-Allow-Origin",
			"http://localhost:5173",
		)
		c.Writer.Header().Set(
			"Access-Control-Allow-Headers",
			"Content-Type, Authorization",
		)
		c.Writer.Header().Set(
			"Access-Control-Allow-Methods",
			"GET, POST, OPTIONS",
		)

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// Authentication
	r.POST("/api/signup", signup)
	r.POST("/api/login", login)

	// Polls
	r.POST("/api/polls", authMiddleware(), createPoll)
	r.GET("/api/polls/:id", getPoll)
	r.GET("/api/polls/:id/results", getResults)
	r.POST("/api/polls/:id/vote", vote)

	// Live updates
	r.GET("/api/polls/:id/live", liveResults)

	fmt.Println("=================================")
	fmt.Println("🚀 LivePoll backend running")
	fmt.Println("👉 http://localhost:8080")
	fmt.Println("=================================")

	r.Run(":8080")
}

// ---------------- SIGNUP ----------------

func signup(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "Invalid input"})
		return
	}

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	if input.Email == "" || len(input.Password) < 6 {
		c.JSON(400, gin.H{
			"error": "Valid email and password of at least 6 characters required",
		})
		return
	}

	count, _ := db.Collection("users").CountDocuments(
		ctx,
		bson.M{"email": input.Email},
	)

	if count > 0 {
		c.JSON(409, gin.H{"error": "User already exists"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(input.Password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		c.JSON(500, gin.H{"error": "Could not create account"})
		return
	}

	user := User{
		Email:    input.Email,
		Password: string(hashedPassword),
	}

	_, err = db.Collection("users").InsertOne(ctx, user)

	if err != nil {
		c.JSON(500, gin.H{"error": "Could not save user"})
		return
	}

	c.JSON(201, gin.H{
		"message": "Account created successfully",
	})
}

// ---------------- LOGIN ----------------

func login(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "Invalid input"})
		return
	}

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	var user User

	err := db.Collection("users").
		FindOne(ctx, bson.M{"email": input.Email}).
		Decode(&user)

	if err != nil {
		c.JSON(401, gin.H{"error": "Invalid email or password"})
		return
	}

	if bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(input.Password),
	) != nil {
		c.JSON(401, gin.H{"error": "Invalid email or password"})
		return
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		Claims{
			UserID: user.ID.Hex(),
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(
					time.Now().Add(24 * time.Hour),
				),
			},
		},
	)

	signedToken, err := token.SignedString(
		[]byte(os.Getenv("JWT_SECRET")),
	)

	if err != nil {
		c.JSON(500, gin.H{"error": "Could not create token"})
		return
	}

	c.JSON(200, gin.H{
		"token": signedToken,
	})
}

// ---------------- AUTH MIDDLEWARE ----------------

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		header := c.GetHeader("Authorization")

		if !strings.HasPrefix(header, "Bearer ") {
			c.JSON(401, gin.H{"error": "Login required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(header, "Bearer ")

		token, err := jwt.ParseWithClaims(
			tokenString,
			&Claims{},
			func(token *jwt.Token) (interface{}, error) {
				return []byte(os.Getenv("JWT_SECRET")), nil
			},
		)

		if err != nil || !token.Valid {
			c.JSON(401, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*Claims)

		if !ok {
			c.JSON(401, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)

		c.Next()
	}
}

// ---------------- CREATE POLL ----------------

func createPoll(c *gin.Context) {

	var input struct {
		Question string   `json:"question"`
		Options  []string `json:"options"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "Invalid input"})
		return
	}

	input.Question = strings.TrimSpace(input.Question)

	if len(input.Question) < 3 {
		c.JSON(400, gin.H{"error": "Question is too short"})
		return
	}

	if len(input.Options) < 2 {
		c.JSON(400, gin.H{
			"error": "At least 2 options are required",
		})
		return
	}

	if len(input.Options) > 6 {
		c.JSON(400, gin.H{
			"error": "Maximum 6 options allowed",
		})
		return
	}

	optionsList := []PollOption{}

	for i, text := range input.Options {

		text = strings.TrimSpace(text)

		if text == "" {
			c.JSON(400, gin.H{
				"error": "Options cannot be empty",
			})
			return
		}

		optionsList = append(optionsList, PollOption{
			ID:   fmt.Sprintf("option-%d", i+1),
			Text: text,
		})
	}

	poll := Poll{
		Question:  input.Question,
		Options:   optionsList,
		CreatedBy: c.GetString("userID"),
		CreatedAt: time.Now(),
	}

	result, err := db.Collection("polls").InsertOne(ctx, poll)

	if err != nil {
		c.JSON(500, gin.H{"error": "Could not create poll"})
		return
	}

	poll.ID = result.InsertedID.(bson.ObjectID)

	// Create Redis counters
	for _, option := range poll.Options {

		rdb.HSet(
			ctx,
			"poll:"+poll.ID.Hex()+":votes",
			option.ID,
			0,
		)
	}

	c.JSON(201, poll)
}

// ---------------- GET POLL ----------------

func getPoll(c *gin.Context) {

	id, err := bson.ObjectIDFromHex(c.Param("id"))

	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid poll ID"})
		return
	}

	var poll Poll

	err = db.Collection("polls").
		FindOne(ctx, bson.M{"_id": id}).
		Decode(&poll)

	if err != nil {
		c.JSON(404, gin.H{"error": "Poll not found"})
		return
	}

	c.JSON(200, poll)
}

// ---------------- GET RESULTS ----------------

func getResults(c *gin.Context) {

	pollID := c.Param("id")

	values, err := rdb.HGetAll(
		ctx,
		"poll:"+pollID+":votes",
	).Result()

	if err != nil {
		c.JSON(500, gin.H{"error": "Could not get results"})
		return
	}

	c.JSON(200, values)
}

// ---------------- VOTE ----------------

func vote(c *gin.Context) {

	pollID := c.Param("id")

	id, err := bson.ObjectIDFromHex(pollID)

	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid poll ID"})
		return
	}

	var input struct {
		OptionID string `json:"optionId"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "Invalid vote"})
		return
	}

	// Check poll exists
	var poll Poll

	err = db.Collection("polls").
		FindOne(ctx, bson.M{"_id": id}).
		Decode(&poll)

	if err != nil {
		c.JSON(404, gin.H{"error": "Poll not found"})
		return
	}

	// Validate option
	valid := false

	for _, option := range poll.Options {
		if option.ID == input.OptionID {
			valid = true
			break
		}
	}

	if !valid {
		c.JSON(400, gin.H{"error": "Invalid option"})
		return
	}

	// Redis increments the live count
	count, err := rdb.HIncrBy(
		ctx,
		"poll:"+pollID+":votes",
		input.OptionID,
		1,
	).Result()

	if err != nil {
		c.JSON(500, gin.H{"error": "Could not record vote"})
		return
	}

	// Store vote in MongoDB
	_, err = db.Collection("votes").InsertOne(
		ctx,
		bson.M{
			"pollId":    id,
			"optionId":  input.OptionID,
			"createdAt": time.Now(),
		},
	)

	if err != nil {
		c.JSON(500, gin.H{"error": "Could not save vote"})
		return
	}

	// Redis Pub/Sub sends live update
	message := fmt.Sprintf(
		`{"optionId":"%s","count":%d}`,
		input.OptionID,
		count,
	)

	rdb.Publish(
		ctx,
		"poll:"+pollID,
		message,
	)

	c.JSON(200, gin.H{
		"message": "Vote recorded",
		"count":   count,
	})
}

// ---------------- LIVE RESULTS ----------------

func liveResults(c *gin.Context) {

	pollID := c.Param("id")

	pubsub := rdb.Subscribe(
		ctx,
		"poll:"+pollID,
	)

	defer pubsub.Close()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	for {

		message, err := pubsub.ReceiveMessage(c)

		if err != nil {
			return
		}

		fmt.Fprintf(
			c.Writer,
			"data: %s\n\n",
			message.Payload,
		)

		c.Writer.Flush()
	}
}
