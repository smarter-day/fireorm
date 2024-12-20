package main

import (
	"cloud.google.com/go/firestore"
	"context"
	"github.com/smarter-day/fireorm"
	"log"
)

type Example struct {
	ID    string `firestore:"id"`
	Name  string `firestore:"name"`
	Email string `firestore:"email"`
}

func createFirestoreClient() *firestore.Client {
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, "smarter-staging")
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	return client
}

func main() {
	ctx := context.Background()
	client := createFirestoreClient()
	defer client.Close()

	connection := fireorm.NewConnection(client)
	db := fireorm.New(connection).Model(&Example{})

	// Create a few users
	users := []Example{
		{Name: "John Doe", Email: "john.doe@example.com"},
		{Name: "Jane Smith", Email: "john.doe2@example.com"},
	}

	for i, user := range users {
		err := db.Save(ctx, &user)
		if err != nil {
			log.Fatalf("Failed to save user %d: %v", i+1, err)
		}
		log.Printf("Example saved: %+v", user)
		users[i] = user // Update local copy with generated ID
	}

	// Retrieve a user by ID
	retrievedUser := &Example{ID: users[0].ID}
	err := db.GetByID(ctx, retrievedUser)
	if err != nil {
		log.Fatalf("Failed to retrieve user by ID: %v", err)
	}
	log.Printf("Retrieved user by ID: %+v", retrievedUser)

	// Find one user by query
	query := []fireorm.Query{
		{Where: []fireorm.WhereClause{
			{Field: "email", Operator: "==", Value: "john.doe@example.com"},
		}},
	}
	foundUser := &Example{}
	err = db.FindOne(ctx, query, foundUser)
	if err != nil {
		log.Fatalf("Failed to find user by query: %v", err)
	}
	log.Printf("Found user by query: %+v", foundUser)

	// Find all users
	var allUsers []Example
	err = db.FindAll(ctx, nil, &allUsers)
	if err != nil {
		log.Fatalf("Failed to retrieve all users: %v", err)
	}
	log.Printf("All users: %+v", allUsers)

	// Update a user's name
	updates := []firestore.Update{
		{Path: "name", Value: "Johnathan Doe"},
	}
	err = db.Update(ctx, &Example{ID: users[0].ID}, updates)
	if err != nil {
		log.Fatalf("Failed to update user: %v", err)
	}
	log.Printf("Updated user ID %s", users[0].ID)

	// Verify update
	updatedUser := &Example{ID: users[0].ID}
	err = db.GetByID(ctx, updatedUser)
	if err != nil {
		log.Fatalf("Failed to verify updated user: %v", err)
	}
	log.Printf("Verified updated user: %+v", updatedUser)

	// Delete all users
	for _, user := range allUsers {
		err = db.Delete(ctx, &user)
		if err != nil {
			log.Fatalf("Failed to delete user ID %s: %v", user.ID, err)
		}
		log.Printf("Deleted user ID %s", user.ID)
	}

	// Verify all users deleted
	allUsers = []Example{}
	err = db.FindAll(ctx, nil, &allUsers)
	if err != nil {
		log.Fatalf("Failed to verify user deletion: %v", err)
	}
	if len(allUsers) == 0 {
		log.Println("All users successfully deleted.")
	} else {
		log.Printf("Some users were not deleted: %+v", allUsers)
	}
}
