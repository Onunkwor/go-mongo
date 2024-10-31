package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/onunkwor/go-mongo/graph/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DB struct {
	client *mongo.Client
	database *mongo.Database
}

type mongoJobListing struct {
    ID          primitive.ObjectID `bson:"_id"`
    Title       string             `bson:"title"`
    Description string             `bson:"description"`
    Company     string             `bson:"company"`
    URL         string             `bson:"url"`
}
// Connect initializes a new MongoDB client and connects to the server.
func Connect(dbName string) *DB {
	// Create a context with a timeout to ensure the connection doesn’t hang indefinitely.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	err := godotenv.Load()
    if err != nil {
        log.Fatalf("Error loading .env file")
    }
	username := os.Getenv("MONGODB_USERNAME")
    password := os.Getenv("MONGODB_PASSWORD")
	connectStr := fmt.Sprintf("mongodb+srv://%s:%s@cluster0.dj3nz.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0", username, password)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectStr))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Ping the database to ensure the connection is established.
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Could not ping MongoDB server: %v", err)
	}
	database := client.Database(dbName)
	log.Println("Connected to MongoDB!")
	return &DB{client: client, database: database}
};

// func (db *DB) CreateJobListing(input *model.CreateJobListingInput) (*model.JobListing, error) {

// }
func (db *DB) GetJobs() ([]*model.JobListing, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	var jobs []*mongoJobListing
	cursor, err := db.database.Collection("jobs").Find(ctx,bson.M{})
	if err != nil {
		return nil,err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var job mongoJobListing
		if err := cursor.Decode(&job); err != nil {
			log.Println("Failed to decode job:", err)
			continue
		}
		jobs = append(jobs, &job)
	}

	// Check for errors encountered during iteration.
	if err := cursor.Err(); err != nil {
		return nil, err
	}
 // Prepare the final result as []*model.JobListing
 result := make([]*model.JobListing, len(jobs))
 for i, mongoJob := range jobs {
	 result[i] = &model.JobListing{
		 ID:          mongoJob.ID.Hex(), // Convert ObjectID to string
		 Title:       mongoJob.Title,
		 Description: mongoJob.Description,
		 Company:     mongoJob.Company,
		 URL:         mongoJob.URL,
	 }
 }

 return result, nil
}

func (db *DB) GetJob(id string) (*model.JobListing, error) {
	jobID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var mongoJob mongoJobListing
	err = db.database.Collection("jobs").FindOne(ctx, bson.M{"_id": jobID}).Decode(&mongoJob)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("job with ID %s not found", id)
		}
		return nil, fmt.Errorf("failed to find job: %w", err)
	}
    job := &model.JobListing{
        ID:          mongoJob.ID.Hex(),
        Title:       mongoJob.Title,
        Description: mongoJob.Description,
        Company:     mongoJob.Company,
        URL:         mongoJob.URL,
    }
    return job, nil

}

func (db *DB) CreateJobListing(input model.CreateJobListingInput) (*model.JobListing, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	jobData := bson.M{
        "title":       input.Title,
        "description": input.Description,
        "company":     input.Company,
        "url":         input.URL,
    }
	result,err := db.database.Collection("jobs").InsertOne(ctx,jobData)
	if err != nil {
		log.Println("Failed to insert job listing:", err)
		return nil, err
	}
	job := &model.JobListing{
        ID:          result.InsertedID.(primitive.ObjectID).Hex(),
        Title:       input.Title,
        Description: input.Description,
        Company:     input.Company,
        URL:         input.URL,
    }
	return job,nil
};

func (db *DB) UpdateJobListing (id string , input model.UpdateJobListingInput) (*model.JobListing, error){
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	// Convert the string ID to a MongoDB ObjectID
	jobID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid job ID: %w", err)
	}
// Create a BSON map to hold the updates
update := bson.M{}
if input.Title != nil {
	update["title"] = *input.Title
}
if input.Description != nil {
	update["description"] = *input.Description
}
if input.Company != nil {
	update["company"] = *input.Company
}
if input.URL != nil {
	update["url"] = *input.URL
}
// Perform the update operation
result := db.database.Collection("jobs").FindOneAndUpdate(ctx, bson.M{"_id": jobID}, bson.M{"$set": update}, options.FindOneAndUpdate().SetReturnDocument(options.After))
if result.Err() != nil {
	if result.Err() == mongo.ErrNoDocuments {
		return nil, fmt.Errorf("job with ID %s not found", id)
	}
	return nil, fmt.Errorf("failed to update job: %w", result.Err())
}
// Create a variable to hold the updated job
var updatedJob model.JobListing
// Decode the result into updatedJob
if err := result.Decode(&updatedJob); err != nil {
	return nil, fmt.Errorf("failed to decode updated job: %w", err)
}

// Convert the ObjectID to string
updatedJob.ID = jobID.Hex()

return &updatedJob, nil
}

func (db *DB) DeleteJobListing(id string) (*model.DeleteJobResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Convert the string ID to a MongoDB ObjectID
	jobID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid job ID: %w", err)
	}

	// Delete the job listing
	result, err := db.database.Collection("jobs").DeleteOne(ctx, bson.M{"_id": jobID})
	if err != nil {
		return nil, fmt.Errorf("failed to delete job: %w", err)
	}

	// Check if a document was deleted
	if result.DeletedCount == 0 {
		return nil, fmt.Errorf("job with ID %s not found", id)
	}

	// Prepare the response
	response := &model.DeleteJobResponse{
		DeleteJobID: id, // Return the ID of the deleted job
	}

	return response, nil
}
