# FireORM

FireORM is an ORM wrapper designed to simplify working with Firebase Firestore in Go. 
It provides transparent mapping between Firestore documents and Go structs, enabling developers to work with Firestore in a more intuitive and efficient manner.

---

## Features

- Transparent handling of Firestore documents as Go structs.
- Connection and transaction management.
- Query builder with `FindOne` and `FindAll` operations.
- CRUD operations: `GetByID`, `Save`, `Update`, `Delete`.
- Customizable collection naming conventions.
- Configurable batch size for efficient bulk updates.
- Transaction support for consistent multi-operation workflows.

---

## Installation

```sh
go get github.com/smarter-day/fireorm
```

---

## Getting Started

### Initialize a Firestore Client

This example demonstrates a standard Firestore client setup. It's provided for beginners and to ensure consistency across subsequent examples.

```go
import (
	"cloud.google.com/go/firestore"
	"context"
	"log"
)

func createFirestoreClient() *firestore.Client {
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, "your-project-id")
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	return client
}
```

### Initialize a FireORM Connection

FireORM provides a connection container that holds the Firestore client and transaction references, allowing you to manage them as needed.

```go
connection := fireorm.NewConnection(createFirestoreClient())
```

---

## Usage Examples

### Define a Model

This is a standard Go struct with `firestore` tags.

The model is named `User`, which by default means the corresponding Firestore collection will be named `users` (an `s` is automatically appended to form the collection name).

```go
type User struct {
    ID    string `firestore:"-"`
    Name  string `firestore:"name"`
    Email string `firestore:"email"`
    Age   int    `firestore:"age"`
}
```

To specify a custom collection name, implement the `CollectionName` method:

```go
func (u User) CollectionName() string {
    return "custom-collection-name"
}
```

### FireORM Initialization

```go
db := fireorm.New(connection).Model(&User{})
```

---

### Operations and Examples

#### Save

`Save` creates or updates a document. If the `ID` field is empty, a new document is created.

```go
user := &User{Name: "John Doe", Email: "john@example.com"}
if err := db.Save(ctx, user); err != nil {
	log.Fatalf("Failed to save user: %v", err)
}
log.Printf("User saved with ID: %s", user.ID)
```

#### GetByID

Retrieve a document by its ID.

```go
retrieved := &User{ID: "user-id"}
if err := db.GetByID(ctx, retrieved); err != nil {
	log.Fatalf("Failed to get user: %v", err)
}
log.Printf("User: %+v", retrieved)
```

#### Update

Update specific fields in a document.

```go
updates := []firestore.Update{{Path: "age", Value: 30}}
if err := db.Update(ctx, &User{ID: "user-id"}, updates); err != nil {
	log.Fatalf("Failed to update user: %v", err)
}
log.Println("User updated successfully")
```

#### Bulk Update

Perform a batch update for documents matching a query.

```go
query := []fireorm.Query{
	{Where: []fireorm.WhereClause{{Field: "age", Operator: ">", Value: 25}}},
}
updates := []firestore.Update{{Path: "status", Value: "active"}}
if err := db.Update(ctx, &User{}, updates, query); err != nil {
	log.Fatalf("Failed to bulk update users: %v", err)
}
log.Println("Bulk update completed successfully")
```

#### Delete

Delete a document by its ID.

```go
user := &User{ID: "user-id"}
if err := db.Delete(ctx, user); err != nil {
	log.Fatalf("Failed to delete user: %v", err)
}
log.Println("User deleted successfully")
```

#### FindOne

Retrieve the first document matching the query.

```go
query := []fireorm.Query{
	{Where: []fireorm.WhereClause{{Field: "email", Operator: "==", Value: "john@example.com"}}},
}
user := &User{}
if err := db.FindOne(ctx, query, user); err != nil {
	log.Fatalf("Failed to find user: %v", err)
}
log.Printf("User: %+v", user)
```

#### FindAll

Retrieve all documents matching a query.

```go
var users []User
if err := db.FindAll(ctx, nil, &users); err != nil {
	log.Fatalf("Failed to find users: %v", err)
}
log.Printf("Users: %+v", users)
```

#### Transactions

Use transactions for atomic operations.

```go
err := connection.GetClient().RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
	dbWithTx := db.WithTransaction(tx)
	user := &User{ID: "user-id"}
	if err := dbWithTx.GetByID(ctx, user); err != nil {
		return err
	}
	user.Name = "Transactional Update"
	return dbWithTx.Save(ctx, user)
})

if err != nil {
	log.Fatalf("Transaction failed: %v", err)
}
log.Println("Transaction completed successfully")
```

---

### Additional Features and Edge Cases

#### Partial Field Save

Save a document with missing fields to test Firestore's zero-value defaults.

```go
user := &User{Name: "Partial User"}
if err := db.Save(ctx, user); err != nil {
	log.Fatalf("Failed to save user: %v", err)
}
```

#### Case Sensitivity

Verify Firestore's case-sensitive query behavior.

```go
query := []fireorm.Query{
	{Where: []fireorm.WhereClause{{Field: "email", Operator: "==", Value: "CASE@EXAMPLE.COM"}}},
}
var results []User
if err := db.FindAll(ctx, query, &results); err != nil {
	log.Fatalf("Failed to query users: %v", err)
}
log.Printf("Results: %+v", results)
```

#### Handling Large Data Sets

Update a large number of documents efficiently using batching.

---

### Testing

To ensure full test coverage, the following tests are provided:
- **CRUD Operations**: Save, GetByID, Update, Delete.
- **Edge Cases**: Missing fields, empty queries, non-existent documents.
- **Batch Updates**: Validate efficient processing of large data sets.
- **Transactions**: Ensure consistency and proper rollback on failure.

For detailed examples, refer to the tests included in the repository.

---

## License

This project is licensed under the MIT License. See the [LICENSE](./LICENSE) file for details.
