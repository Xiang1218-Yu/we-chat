package repository

import (
	"context"
	"log"
	"time"

	"we-chat/internal/config"
	"we-chat/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBRepository struct {
	client   *mongo.Client
	database *mongo.Database
}

func NewMongoDBRepository() *MongoDBRepository {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(config.AppConfig.MongoDB.URI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("Failed to ping MongoDB:", err)
	}

	log.Println("Connected to MongoDB")

	db := client.Database(config.AppConfig.MongoDB.Database)
	repo := &MongoDBRepository{
		client:   client,
		database: db,
	}

	repo.createIndexes()
	return repo
}

func (r *MongoDBRepository) createIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "username", Value: 1}, {Key: "unique", Value: true}}},
		{Keys: bson.D{{Key: "email", Value: 1}, {Key: "unique", Value: true}}},
	}
	r.database.Collection("users").Indexes().CreateMany(ctx, userIndexes)

	roomIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "room_id", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
	}
	r.database.Collection("messages").Indexes().CreateMany(ctx, roomIndexes)
}

func (r *MongoDBRepository) GetCollection(name string) *mongo.Collection {
	return r.database.Collection(name)
}

func (r *MongoDBRepository) Disconnect() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r.client.Disconnect(ctx)
}

// User operations
func (r *MongoDBRepository) CreateUser(ctx context.Context, user *models.User) error {
	collection := r.GetCollection("users")
	user.ID = primitive.NewObjectID()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := collection.InsertOne(ctx, user)
	return err
}

func (r *MongoDBRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	collection := r.GetCollection("users")
	var user models.User
	err := collection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *MongoDBRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	collection := r.GetCollection("users")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var user models.User
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *MongoDBRepository) UpdateUserStatus(ctx context.Context, userID string, status models.UserStatus) error {
	collection := r.GetCollection("users")
	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	_, err = collection.UpdateByID(ctx, objID, update)
	return err
}

// Message operations
func (r *MongoDBRepository) SaveMessage(ctx context.Context, message *models.Message) error {
	collection := r.GetCollection("messages")
	message.ID = primitive.NewObjectID()
	message.CreatedAt = time.Now()
	message.Read = false

	_, err := collection.InsertOne(ctx, message)
	return err
}

func (r *MongoDBRepository) GetRoomMessages(ctx context.Context, roomID string, limit int64) ([]models.Message, error) {
	collection := r.GetCollection("messages")

	findOptions := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(limit)

	cursor, err := collection.Find(ctx, bson.M{"room_id": roomID}, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	if err = cursor.All(ctx, &messages); err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *MongoDBRepository) GetPrivateMessages(ctx context.Context, userID1, userID2 string, limit int64) ([]models.Message, error) {
	collection := r.GetCollection("messages")

	filter := bson.M{
		"$or": []bson.M{
			{"sender_id": userID1, "receiver_id": userID2},
			{"sender_id": userID2, "receiver_id": userID1},
		},
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(limit)

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	if err = cursor.All(ctx, &messages); err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *MongoDBRepository) MarkMessageAsRead(ctx context.Context, messageID string) error {
	collection := r.GetCollection("messages")
	objID, err := primitive.ObjectIDFromHex(messageID)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"read":     true,
			"read_at":  time.Now(),
		},
	}

	_, err = collection.UpdateByID(ctx, objID, update)
	return err
}

func (r *MongoDBRepository) AddReaction(ctx context.Context, messageID string, reaction models.Reaction) error {
	collection := r.GetCollection("messages")
	objID, err := primitive.ObjectIDFromHex(messageID)
	if err != nil {
		return err
	}

	update := bson.M{
		"$push": bson.M{"reactions": reaction},
	}

	_, err = collection.UpdateByID(ctx, objID, update)
	return err
}

// Room operations
func (r *MongoDBRepository) CreateRoom(ctx context.Context, room *models.Room) error {
	collection := r.GetCollection("rooms")
	room.ID = primitive.NewObjectID()
	room.CreatedAt = time.Now()
	room.UpdatedAt = time.Now()

	_, err := collection.InsertOne(ctx, room)
	return err
}

func (r *MongoDBRepository) GetRoomByID(ctx context.Context, id string) (*models.Room, error) {
	collection := r.GetCollection("rooms")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var room models.Room
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&room)
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *MongoDBRepository) GetUserRooms(ctx context.Context, userID string) ([]models.Room, error) {
	collection := r.GetCollection("rooms")

	filter := bson.M{"members": userID}
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var rooms []models.Room
	if err = cursor.All(ctx, &rooms); err != nil {
		return nil, err
	}

	return rooms, nil
}

func (r *MongoDBRepository) AddMemberToRoom(ctx context.Context, roomID, userID string) error {
	collection := r.GetCollection("rooms")
	objID, err := primitive.ObjectIDFromHex(roomID)
	if err != nil {
		return err
	}

	update := bson.M{
		"$push": bson.M{"members": userID},
		"$set":  bson.M{"updated_at": time.Now()},
	}

	_, err = collection.UpdateByID(ctx, objID, update)
	return err
}

func (r *MongoDBRepository) RemoveMemberFromRoom(ctx context.Context, roomID, userID string) error {
	collection := r.GetCollection("rooms")
	objID, err := primitive.ObjectIDFromHex(roomID)
	if err != nil {
		return err
	}

	update := bson.M{
		"$pull": bson.M{"members": userID},
		"$set":  bson.M{"updated_at": time.Now()},
	}

	_, err = collection.UpdateByID(ctx, objID, update)
	return err
}