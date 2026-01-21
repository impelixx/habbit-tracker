package db

import (
	"context"
	"fmt"
	"time"

	"github.com/impelixx/habbit-tracker/backend/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB wraps MongoDB client and provides database operations
type MongoDB struct {
	client   *mongo.Client
	database *mongo.Database
}

// Connect establishes connection to MongoDB
func Connect(uri, database string) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(10).
		SetMinPoolSize(2)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := &MongoDB{
		client:   client,
		database: client.Database(database),
	}

	// Create indexes
	if err := db.createIndexes(ctx); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return db, nil
}

// Close closes the MongoDB connection
func (db *MongoDB) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return db.client.Disconnect(ctx)
}

// createIndexes creates necessary indexes for collections
func (db *MongoDB) createIndexes(ctx context.Context) error {
	// Users collection indexes
	usersCollection := db.database.Collection("users")
	_, err := usersCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "telegramId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create users indexes: %w", err)
	}

	// Tasks collection indexes
	tasksCollection := db.database.Collection("tasks")
	_, err = tasksCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "userId", Value: 1}, {Key: "dueDate", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "userId", Value: 1}, {Key: "completed", Value: 1}},
		},
		{
			Keys:    bson.D{{Key: "completedAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(7776000), // 90 days TTL
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create tasks indexes: %w", err)
	}

	// Reminders collection indexes
	remindersCollection := db.database.Collection("reminders")
	_, err = remindersCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "userId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "enabled", Value: 1}, {Key: "time", Value: 1}},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create reminders indexes: %w", err)
	}

	return nil
}

// User operations

// CreateUser creates a new user
func (db *MongoDB) CreateUser(ctx context.Context, user *models.User) error {
	collection := db.database.Collection("users")
	_, err := collection.InsertOne(ctx, user)
	return err
}

// GetUserByTelegramID retrieves a user by Telegram ID
func (db *MongoDB) GetUserByTelegramID(ctx context.Context, telegramID int64) (*models.User, error) {
	collection := db.database.Collection("users")
	var user models.User
	err := collection.FindOne(ctx, bson.M{"telegramId": telegramID}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID retrieves a user by ID
func (db *MongoDB) GetUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	collection := db.database.Collection("users")
	var user models.User
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUser updates a user
func (db *MongoDB) UpdateUser(ctx context.Context, user *models.User) error {
	collection := db.database.Collection("users")
	user.UpdatedAt = time.Now()
	_, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": user.ID},
		bson.M{"$set": user},
	)
	return err
}

// Task operations

// CreateTask creates a new task
func (db *MongoDB) CreateTask(ctx context.Context, task *models.Task) error {
	collection := db.database.Collection("tasks")
	_, err := collection.InsertOne(ctx, task)
	return err
}

// GetTaskByID retrieves a task by ID
func (db *MongoDB) GetTaskByID(ctx context.Context, id primitive.ObjectID) (*models.Task, error) {
	collection := db.database.Collection("tasks")
	var task models.Task
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&task)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetTasksByUserID retrieves all tasks for a user
func (db *MongoDB) GetTasksByUserID(ctx context.Context, userID primitive.ObjectID) ([]*models.Task, error) {
	collection := db.database.Collection("tasks")
	cursor, err := collection.Find(ctx, bson.M{"userId": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tasks []*models.Task
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// GetTasksByUserIDAndDate retrieves tasks for a user on a specific date
func (db *MongoDB) GetTasksByUserIDAndDate(ctx context.Context, userID primitive.ObjectID, date time.Time) ([]*models.Task, error) {
	collection := db.database.Collection("tasks")

	// Get start and end of the day
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	filter := bson.M{
		"userId": userID,
		"dueDate": bson.M{
			"$gte": startOfDay,
			"$lt":  endOfDay,
		},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tasks []*models.Task
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// UpdateTask updates a task
func (db *MongoDB) UpdateTask(ctx context.Context, task *models.Task) error {
	collection := db.database.Collection("tasks")
	task.UpdatedAt = time.Now()
	_, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": task.ID},
		bson.M{"$set": task},
	)
	return err
}

// DeleteTask deletes a task
func (db *MongoDB) DeleteTask(ctx context.Context, id primitive.ObjectID) error {
	collection := db.database.Collection("tasks")
	_, err := collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// Reminder operations

// CreateReminder creates a new reminder
func (db *MongoDB) CreateReminder(ctx context.Context, reminder *models.Reminder) error {
	collection := db.database.Collection("reminders")
	_, err := collection.InsertOne(ctx, reminder)
	return err
}

// GetReminderByUserID retrieves a reminder by user ID
func (db *MongoDB) GetReminderByUserID(ctx context.Context, userID primitive.ObjectID) (*models.Reminder, error) {
	collection := db.database.Collection("reminders")
	var reminder models.Reminder
	err := collection.FindOne(ctx, bson.M{"userId": userID}).Decode(&reminder)
	if err != nil {
		return nil, err
	}
	return &reminder, nil
}

// UpdateReminder updates a reminder
func (db *MongoDB) UpdateReminder(ctx context.Context, reminder *models.Reminder) error {
	collection := db.database.Collection("reminders")
	reminder.UpdatedAt = time.Now()
	_, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": reminder.ID},
		bson.M{"$set": reminder},
	)
	return err
}

// UpsertReminder creates or updates a reminder
func (db *MongoDB) UpsertReminder(ctx context.Context, reminder *models.Reminder) error {
	collection := db.database.Collection("reminders")
	reminder.UpdatedAt = time.Now()

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(
		ctx,
		bson.M{"userId": reminder.UserID},
		bson.M{"$set": reminder},
		opts,
	)
	return err
}

// GetEnabledReminders retrieves all enabled reminders
func (db *MongoDB) GetEnabledReminders(ctx context.Context) ([]*models.Reminder, error) {
	collection := db.database.Collection("reminders")
	cursor, err := collection.Find(ctx, bson.M{"enabled": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var reminders []*models.Reminder
	if err := cursor.All(ctx, &reminders); err != nil {
		return nil, err
	}
	return reminders, nil
}
