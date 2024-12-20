package tests

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/smarter-day/fireorm"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/iterator"
)

type User struct {
	ID    string `firestore:"-"`
	Name  string `firestore:"name"`
	Email string `firestore:"email"`
	Age   int    `firestore:"age"`
}

func startFirestoreEmulator() *exec.Cmd {
	cmd := exec.Command("firebase", "emulators:start", "--only", "firestore")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start Firestore emulator: %v", err)
	}

	log.Println("Waiting for Firestore emulator to initialize...")
	// Allow emulator to initialize
	return cmd
}

func stopFirestoreEmulator(cmd *exec.Cmd) {
	if err := cmd.Process.Kill(); err != nil {
		log.Fatalf("Failed to stop Firestore emulator: %v", err)
	}
}

func createFirestoreClient() *firestore.Client {
	ctx := context.Background()
	os.Setenv("FIRESTORE_EMULATOR_HOST", "localhost:8080")
	client, err := firestore.NewClient(ctx, "test-project")
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	return client
}

func resetFirestoreEmulator(ctx context.Context, client *firestore.Client) {
	collections := []string{"users"}
	for _, collection := range collections {
		iter := client.Collection(collection).Documents(ctx)
		for {
			doc, err := iter.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				log.Fatalf("Failed to iterate documents: %v", err)
			}
			_, err = doc.Ref.Delete(ctx)
			if err != nil {
				log.Fatalf("Failed to delete document: %v", err)
			}
		}
	}
}

func TestFireORM(t *testing.T) {
	emulator := startFirestoreEmulator()
	defer stopFirestoreEmulator(emulator)

	ctx := context.Background()
	client := createFirestoreClient()
	defer client.Close()

	connection := fireorm.NewConnection(client)
	db := fireorm.New(connection).Model(&User{})

	// Reset emulator state before running tests
	resetFirestoreEmulator(ctx, client)

	t.Run("Save and Retrieve", func(t *testing.T) {
		user := &User{Name: "John Doe", Email: "john.doe@example.com", Age: 30}
		err := db.Save(ctx, user)
		assert.NoError(t, err)
		assert.NotEmpty(t, user.ID)

		retrieved := &User{ID: user.ID}
		err = db.GetByID(ctx, retrieved)
		assert.NoError(t, err)
		assert.Equal(t, user.Name, retrieved.Name)
		assert.Equal(t, user.Email, retrieved.Email)
		assert.Equal(t, user.Age, retrieved.Age)
	})

	t.Run("Update by ID", func(t *testing.T) {
		user := &User{Name: "Jane Doe", Email: "jane.doe@example.com", Age: 25}
		err := db.Save(ctx, user)
		assert.NoError(t, err)

		updates := []firestore.Update{
			{Path: "age", Value: 26},
		}
		err = db.Update(ctx, &User{ID: user.ID}, updates)
		assert.NoError(t, err)

		retrieved := &User{ID: user.ID}
		err = db.GetByID(ctx, retrieved)
		assert.NoError(t, err)
		assert.Equal(t, 26, retrieved.Age)
	})

	t.Run("Bulk Update by Query", func(t *testing.T) {
		users := []User{
			{Name: "Alice", Email: "alice@example.com", Age: 35},
			{Name: "Bob", Email: "bob@example.com", Age: 40},
		}
		for _, user := range users {
			err := db.Save(ctx, &user)
			assert.NoError(t, err)
		}

		query := []fireorm.Query{
			{
				Where: []fireorm.WhereClause{
					{Field: "age", Operator: ">", Value: 30},
				},
			},
		}
		updates := []firestore.Update{
			{Path: "age", Value: 50},
		}
		err := db.Update(ctx, &User{}, updates, query)
		assert.NoError(t, err)

		var updatedUsers []User
		err = db.FindAll(ctx, query, &updatedUsers)
		assert.NoError(t, err)
		for _, user := range updatedUsers {
			assert.Equal(t, 50, user.Age)
		}
	})

	t.Run("FindOne and FindAll", func(t *testing.T) {
		query := []fireorm.Query{
			{
				Where: []fireorm.WhereClause{
					{Field: "email", Operator: "==", Value: "john.doe@example.com"},
				},
			},
		}
		found := &User{}
		err := db.FindOne(ctx, query, found)
		assert.NoError(t, err)
		assert.Equal(t, "john.doe@example.com", found.Email)

		var allUsers []User
		err = db.FindAll(ctx, nil, &allUsers)
		assert.NoError(t, err)
		assert.NotEmpty(t, allUsers)
	})

	t.Run("Delete by ID", func(t *testing.T) {
		user := &User{Name: "To Be Deleted", Email: "delete.me@example.com", Age: 20}
		err := db.Save(ctx, user)
		assert.NoError(t, err)

		err = db.Delete(ctx, user)
		assert.NoError(t, err)

		retrieved := &User{ID: user.ID}
		err = db.GetByID(ctx, retrieved)
		assert.Error(t, err, "Expected an error for a non-existent document")
	})

	t.Run("Save with Partial Fields", func(t *testing.T) {
		user := &User{Name: "Partial Fields"}
		err := db.Save(ctx, user)
		assert.NoError(t, err)
		assert.NotEmpty(t, user.ID)

		retrieved := &User{ID: user.ID}
		err = db.GetByID(ctx, retrieved)
		assert.NoError(t, err)
		assert.Equal(t, "Partial Fields", retrieved.Name)
		assert.Equal(t, "", retrieved.Email)
		assert.Equal(t, 0, retrieved.Age) // Default zero value
	})

	t.Run("Update Non-Existent Document", func(t *testing.T) {
		updates := []firestore.Update{
			{Path: "age", Value: 100},
		}
		err := db.Update(ctx, &User{ID: "non-existent-id"}, updates)
		assert.Error(t, err, "Expected an error for updating a non-existent document")
	})

	t.Run("Delete Non-Existent Document", func(t *testing.T) {
		err := db.Delete(ctx, &User{ID: "non-existent-id"})
		assert.NoError(t, err, "Deleting a non-existent document should not cause an error")
	})

	t.Run("Query with No Results", func(t *testing.T) {
		query := []fireorm.Query{
			{
				Where: []fireorm.WhereClause{
					{Field: "age", Operator: ">", Value: 1000},
				},
			},
		}
		var results []User
		err := db.FindAll(ctx, query, &results)
		assert.NoError(t, err)
		assert.Empty(t, results, "Expected no results for the query")
	})

	t.Run("Transaction Test", func(t *testing.T) {
		err := client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			dbWithTx := db.WithTransaction(tx)
			user := &User{Name: "Transactional User", Email: "tx.user@example.com", Age: 29}
			err := dbWithTx.Save(ctx, user)
			if err != nil {
				return err
			}
			// Simulate an error to test rollback
			return errors.New("simulate transaction failure")
		})
		assert.Error(t, err, "Transaction should fail and rollback changes")

		// Verify the user was not saved
		query := []fireorm.Query{
			{
				Where: []fireorm.WhereClause{
					{Field: "email", Operator: "==", Value: "tx.user@example.com"},
				},
			},
		}
		var results []User
		err = db.FindAll(ctx, query, &results)
		assert.NoError(t, err)
		assert.Empty(t, results, "User should not be saved due to transaction rollback")
	})

	t.Run("Batch Save", func(t *testing.T) {
		users := []User{
			{Name: "Batch User 1", Email: "batch1@example.com"},
			{Name: "Batch User 2", Email: "batch2@example.com"},
		}
		for _, user := range users {
			err := db.Save(ctx, &user)
			assert.NoError(t, err)
			assert.NotEmpty(t, user.ID)
		}

		// Verify all users are saved
		var savedUsers []User
		err := db.FindAll(ctx, nil, &savedUsers)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(savedUsers), len(users), "All batch-saved users should exist")
	})

	t.Run("Case Sensitivity in Queries", func(t *testing.T) {
		user := &User{Name: "CaseSensitive", Email: "case@example.com"}
		err := db.Save(ctx, user)
		assert.NoError(t, err)

		query := []fireorm.Query{
			{
				Where: []fireorm.WhereClause{
					{Field: "email", Operator: "==", Value: "CASE@EXAMPLE.COM"},
				},
			},
		}
		var results []User
		err = db.FindAll(ctx, query, &results)
		assert.NoError(t, err)
		assert.Empty(t, results, "Firestore queries are case-sensitive by default")
	})

	t.Run("Large Data Set Update", func(t *testing.T) {
		// Create a large data set
		for i := 0; i < 1000; i++ {
			user := &User{Name: fmt.Sprintf("User %d", i), Email: fmt.Sprintf("user%d@example.com", i), Age: 20}
			err := db.Save(ctx, user)
			assert.NoError(t, err)
		}

		query := []fireorm.Query{
			{
				Where: []fireorm.WhereClause{
					{Field: "age", Operator: "==", Value: 20},
				},
			},
		}
		updates := []firestore.Update{
			{Path: "age", Value: 30},
		}
		err := db.Update(ctx, &User{}, updates, query)
		assert.NoError(t, err)

		var updatedUsers []User
		err = db.FindAll(ctx, query, &updatedUsers)
		assert.NoError(t, err)
		assert.Empty(t, updatedUsers, "No users should have age 20 after the update")
	})

	t.Run("Field Collision", func(t *testing.T) {
		user := &User{Name: "Field Collision", Email: "collision@example.com", Age: 99}
		err := db.Save(ctx, user)
		assert.NoError(t, err)

		retrieved := &User{ID: user.ID}
		err = db.GetByID(ctx, retrieved)
		assert.NoError(t, err)
		assert.Equal(t, user.Name, retrieved.Name)
		assert.Equal(t, user.Email, retrieved.Email)
	})
}
