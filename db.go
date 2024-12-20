package fireorm

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"reflect"
	"strings"
)

// IDB defines the interface for database operations.
type IDB interface {
	Model(interface{}) IDB
	WithConnection(connection IConnection) IDB
	WithTransaction(tx *firestore.Transaction) IDB
	CollectionName() (string, error)
	GetByID(ctx context.Context, model interface{}) error
	FindOne(ctx context.Context, queries []Query, dest interface{}) error
	FindAll(ctx context.Context, queries []Query, dest interface{}) error
	ApplyQueries(ctx context.Context, q firestore.Query, queries []Query) (firestore.Query, error)
	Save(ctx context.Context, model interface{}, fieldsToSave ...string) error
	Update(ctx context.Context, model interface{}, updates []firestore.Update, where ...[]Query) error
	Delete(ctx context.Context, model interface{}) error
	GetID(model interface{}) string
	GetModelType() reflect.Type
	GetModelValue() reflect.Value
	SetUpdateBatchSize(size int) IDB
	GetUpdateBatchSize() int
	GetConnection() IConnection
	SetConnection(conn IConnection) IDB
}

type dbOptions struct {
	conn            IConnection
	modelType       reflect.Type
	modelVal        reflect.Value
	updateBatchSize int
}

// DB holds the Firestore connection and state about the current model.
type DB struct {
	options dbOptions
}

// New initializes a new DB instance.
func New(conn IConnection) IDB {
	return &DB{
		options: dbOptions{
			conn:            conn,
			modelType:       nil,
			modelVal:        reflect.Value{},
			updateBatchSize: 100,
		},
	}
}

// GetConnection returns the Firestore connection associated with the DB instance.
func (db *DB) GetConnection() IConnection {
	return db.options.conn
}

// SetConnection sets the Firestore connection associated with the DB instance.
func (db *DB) SetConnection(conn IConnection) IDB {
	db.options.conn = conn
	return db
}

// WithConnection returns a new DB instance with the specified connection.
func (db *DB) WithConnection(connection IConnection) IDB {
	newInstance := &DB{
		options: db.options,
	}
	newInstance.SetConnection(connection)
	return newInstance
}

// SetUpdateBatchSize sets the size of the update batch.
func (db *DB) SetUpdateBatchSize(size int) IDB {
	newInstance := &DB{
		options: db.options,
	}
	newInstance.options.updateBatchSize = size
	return db
}

// GetUpdateBatchSize returns the size of the update batch.
func (db *DB) GetUpdateBatchSize() int {
	return db.options.updateBatchSize
}

// GetModelType returns the type of the model associated with the DB instance.
func (db *DB) GetModelType() reflect.Type {
	return db.options.modelType
}

// GetModelValue returns the value of the model associated with the DB instance.
func (db *DB) GetModelValue() reflect.Value {
	return db.options.modelVal
}

// WithTransaction returns a new DB instance using the given transaction.
func (db *DB) WithTransaction(tx *firestore.Transaction) IDB {
	newConnection := NewConnection(db.options.conn.GetClient(), tx)
	newInstance := &DB{
		options: db.options,
	}
	newInstance.SetConnection(newConnection)
	return newInstance
}

// Model sets the model type for the DB instance.
// Model should be a struct or a pointer to a struct.
func (db *DB) Model(model interface{}) IDB {
	v := reflect.ValueOf(model)
	t := v.Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		panic("model must be a struct or pointer to a struct")
	}

	newInstance := &DB{
		options: db.options,
	}
	newInstance.options.modelType = t
	newInstance.options.modelVal = reflect.New(t)
	return newInstance
}

// GetByID retrieves a single document by ID and stores it in dest.
func (db *DB) GetByID(ctx context.Context, model interface{}) error {
	getByIdFunc := func(dbInstance *DB) error {
		if dbInstance.GetModelType() == nil {
			return fmt.Errorf("no model set, call db.Model(&Model{}) first")
		}

		colName, err := dbInstance.CollectionName()
		if err != nil {
			return err
		}

		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic occurred: %v", r)
			}
		}()

		id := dbInstance.GetID(model)
		if id == "" {
			return fmt.Errorf("ID cannot be empty")
		}
		docRef := dbInstance.GetConnection().GetClient().Collection(colName).Doc(id)

		var doc *firestore.DocumentSnapshot
		if dbInstance.GetConnection().HasTransaction() {
			doc, err = dbInstance.GetConnection().GetTransaction().Get(docRef)
		} else {
			doc, err = docRef.Get(ctx)
		}
		if err != nil {
			return err
		}

		err = doc.DataTo(&model)
		if err != nil {
			return fmt.Errorf("failed to parse document: %v", err)
		}
		return nil
	}
	return getByIdFunc(db.Model(model).(*DB))
}

// CollectionName derives the collection name from the model's type name.
// Customize as needed for your naming conventions.
func (db *DB) CollectionName() (string, error) {
	if db.GetModelType() == nil {
		return "", fmt.Errorf("no model set")
	}

	// Check if the model has a CollectionName() method
	method := db.GetModelValue().MethodByName("CollectionName")
	if method.IsValid() && method.Type().NumIn() == 0 && method.Type().NumOut() == 1 && method.Type().Out(0).Kind() == reflect.String {
		results := method.Call(nil)
		collectionName, ok := results[0].Interface().(string)
		if !ok {
			return "", fmt.Errorf("CollectionName method does not return a string")
		}
		return collectionName, nil
	}

	// Default: use the lowercased type name + "s"
	return strings.ToLower(db.GetModelType().Name()) + "s", nil
}

// FindAll retrieves multiple documents based on queries and stores them in dest (which must be a pointer to a slice).
func (db *DB) FindAll(ctx context.Context, queries []Query, dest interface{}) error {
	findAll := func(dbInstance *DB) error {
		if dbInstance.GetModelType() == nil {
			return fmt.Errorf("no model set, call db.Model(&Model{}) first")
		}

		colName, err := dbInstance.CollectionName()
		if err != nil {
			return err
		}

		q := dbInstance.GetConnection().GetClient().Collection(colName).Query

		if queries != nil && len(queries) != 0 {
			q, err = dbInstance.ApplyQueries(ctx, q, queries)
			if err != nil {
				return err
			}
		}

		// Handle transaction or no transaction
		var docs []*firestore.DocumentSnapshot
		if dbInstance.GetConnection().HasTransaction() {
			docs, err = dbInstance.GetConnection().GetTransaction().Documents(q).GetAll()
		} else {
			docs, err = q.Documents(ctx).GetAll()
		}
		if err != nil {
			return err
		}

		rv := reflect.ValueOf(dest)
		if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Slice {
			return fmt.Errorf("dest must be a pointer to a slice")
		}

		sliceVal := rv.Elem()
		for _, doc := range docs {
			newInstance := reflect.New(dbInstance.GetModelType()).Interface()
			if err := doc.DataTo(newInstance); err != nil {
				return fmt.Errorf("failed to parse document: %v", err)
			}
			SetIDField(newInstance, doc.Ref.ID)
			sliceVal = reflect.Append(sliceVal, reflect.ValueOf(newInstance).Elem())
		}
		rv.Elem().Set(sliceVal)
		return nil
	}
	// Dest is a slice of structs, so check what is the destination type
	destType := reflect.TypeOf(dest).Elem()
	if destType.Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to a slice")
	}
	// Check what is the type of one slice element
	elemType := destType.Elem()
	if elemType.Kind() != reflect.Struct {
		return fmt.Errorf("dest slice element must be a struct")
	}
	elemTypeInstance := reflect.New(elemType).Interface()
	return findAll(db.Model(elemTypeInstance).(*DB))
}

// FindOne retrieves a single document based on queries and stores it in dest (which must be a pointer to a struct).
func (db *DB) FindOne(ctx context.Context, queries []Query, dest interface{}) error {
	findOne := func(dbInstance *DB) error {
		if dbInstance.GetModelType() == nil {
			return fmt.Errorf("no model set, call db.Model(&Model{}) first")
		}

		colName, err := dbInstance.CollectionName()
		if err != nil {
			return err
		}

		q := dbInstance.GetConnection().GetClient().Collection(colName).Query
		q, err = dbInstance.ApplyQueries(ctx, q, queries)
		if err != nil {
			return err
		}

		// Ensure we only get one document
		q = q.Limit(1)

		var docs []*firestore.DocumentSnapshot
		if dbInstance.GetConnection().HasTransaction() {
			docs, err = dbInstance.GetConnection().GetTransaction().Documents(q).GetAll()
		} else {
			docs, err = q.Documents(ctx).GetAll()
		}
		if err != nil {
			return err
		}

		if len(docs) == 0 {
			return fmt.Errorf("no document found")
		}

		if err := docs[0].DataTo(dest); err != nil {
			return fmt.Errorf("failed to parse document: %v", err)
		}
		SetIDField(dest, docs[0].Ref.ID)
		return nil
	}
	return findOne(db.Model(dest).(*DB))
}

// Save inserts or updates a document.
// If the model has no ID set and no fieldsToSave are specified, a new document is created.
// If fieldsToSave are specified but no ID is set, returns an error (can't update without ID).
func (db *DB) Save(ctx context.Context, model interface{}, fieldsToSave ...string) error {
	save := func(dbInstance *DB) error {
		if dbInstance.GetModelType() == nil {
			return fmt.Errorf("no model set, call db.Model(&Model{}) first")
		}

		colName, err := dbInstance.CollectionName()
		if err != nil {
			return err
		}

		id := dbInstance.GetID(model)
		docRef := dbInstance.GetConnection().GetClient().Collection(colName).Doc(id)
		data, err := StructToMap(model)
		if err != nil {
			return err
		}

		// If no ID is specified and no fieldsToSave are provided, create a new document
		if id == "" && (fieldsToSave == nil || len(fieldsToSave) == 0) {
			docRef = dbInstance.GetConnection().GetClient().Collection(colName).NewDoc()
			SetIDField(model, docRef.ID)
			id = docRef.ID
		}

		// If fieldsToSave are given but no ID, we cannot update a non-existing doc
		if len(fieldsToSave) > 0 && id == "" {
			return fmt.Errorf("cannot update fields on a record with no ID")
		}

		if len(fieldsToSave) == 0 {
			// Set or create the entire document
			if dbInstance.GetConnection().HasTransaction() {
				return dbInstance.GetConnection().GetTransaction().Set(docRef, data)
			}
			_, err = docRef.Set(ctx, data)
			return err
		}

		// Update selected fields only
		var updates []firestore.Update
		for _, field := range fieldsToSave {
			value, ok := data[field]
			if !ok {
				return fmt.Errorf("field %s not found in model data", field)
			}
			updates = append(updates, firestore.Update{
				Path:  field,
				Value: value,
			})
		}

		if dbInstance.GetConnection().HasTransaction() {
			return dbInstance.GetConnection().GetTransaction().Update(docRef, updates)
		}
		_, err = docRef.Update(ctx, updates)
		return err
	}
	return save(db.Model(model).(*DB))
}

// Update updates the document identified by the model's ID with the provided firestore updates.
func (db *DB) Update(ctx context.Context, model interface{}, updates []firestore.Update, where ...[]Query) error {
	update := func(dbInstance *DB) error {
		if dbInstance.GetModelType() == nil {
			return fmt.Errorf("no model set, call db.Model(&Model{}) first")
		}

		colName, err := dbInstance.CollectionName()
		if err != nil {
			return err
		}

		id := dbInstance.GetID(model)
		if id != "" {
			// Direct update by ID
			docRef := dbInstance.GetConnection().GetClient().Collection(colName).Doc(id)
			if dbInstance.GetConnection().HasTransaction() {
				return dbInstance.GetConnection().GetTransaction().Update(docRef, updates)
			}
			_, err = docRef.Update(ctx, updates)
			return err
		}

		// Update by query if no ID is provided
		if len(where) == 0 || len(where[0]) == 0 {
			return fmt.Errorf("either ID or query conditions must be provided")
		}

		q := dbInstance.GetConnection().GetClient().Collection(colName).Query
		q, err = dbInstance.ApplyQueries(ctx, q, where[0])
		if err != nil {
			return err
		}

		var lastDoc *firestore.DocumentSnapshot

		for {
			// Skip StartAfter for the first iteration
			query := q
			if lastDoc != nil {
				query = q.StartAfter(lastDoc)
			}

			iter := query.Limit(dbInstance.GetUpdateBatchSize()).Documents(ctx)
			docs, err := iter.GetAll()
			if err != nil {
				return fmt.Errorf("failed to retrieve documents: %v", err)
			}

			if len(docs) == 0 {
				break
			}

			batch := dbInstance.GetConnection().GetClient().Batch()
			for _, doc := range docs {
				batch.Update(doc.Ref, updates)
			}

			if dbInstance.GetConnection().HasTransaction() {
				return fmt.Errorf("transactional batch updates are not supported")
			}

			_, err = batch.Commit(ctx)
			if err != nil {
				return fmt.Errorf("batch commit failed: %v", err)
			}

			lastDoc = docs[len(docs)-1] // Update lastDoc for the next iteration
		}

		return nil
	}
	return update(db.Model(model).(*DB))
}

// Delete removes the document identified by the model's ID from Firestore.
func (db *DB) Delete(ctx context.Context, model interface{}) error {
	if db.GetModelType() == nil {
		return fmt.Errorf("no model set, call db.Model(&Model{}) first")
	}

	colName, err := db.CollectionName()
	if err != nil {
		return err
	}

	id := db.GetID(model)
	if id == "" {
		return fmt.Errorf("ID cannot be empty for delete")
	}

	docRef := db.GetConnection().GetClient().Collection(colName).Doc(id)
	if db.GetConnection().HasTransaction() {
		return db.GetConnection().GetTransaction().Delete(docRef)
	}
	_, err = docRef.Delete(ctx)
	return err
}

// ApplyQueries applies the given queries (where, orderBy, limit) to the given Firestore query.
func (db *DB) ApplyQueries(ctx context.Context, q firestore.Query, queries []Query) (firestore.Query, error) {
	for _, qry := range queries {
		for _, w := range qry.Where {
			value := w.Value
			if w.ValueProvider != nil {
				v, err := w.ValueProvider.GetValue(ctx)
				if err != nil {
					return q, fmt.Errorf("failed to get value for field %s: %v", w.Field, err)
				}
				value = v
			}
			q = q.Where(w.Field, w.Operator, value)
		}

		for _, o := range qry.OrderBy {
			q = q.OrderBy(o.Field, o.Direction)
		}

		if qry.Limit > 0 && qry.Limit != QueryLimitUnlimited {
			q = q.Limit(qry.Limit)
		}
	}
	return q, nil
}

// GetID retrieves the "ID" field value if it exists and is a string.
func (db *DB) GetID(model interface{}) string {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	field := v.FieldByName("ID")
	if field.IsValid() && field.Kind() == reflect.String {
		return field.String()
	}
	return ""
}
