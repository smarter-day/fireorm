package fireorm

import (
	"cloud.google.com/go/firestore"
	"context"
)

const (
	QueryLimitMax       = 10_000
	QueryLimitUnlimited = -1
)

type IValueProvider interface {
	GetValue(ctx context.Context) (interface{}, error)
	SaveLastValue(ctx context.Context, change *firestore.DocumentChange) error
}

// Query defines the structure of a Firestore query.
type Query struct {
	Where   []WhereClause
	OrderBy []OrderClause
	Limit   int
}

// WhereClause defines a single where condition.
type WhereClause struct {
	Field         string
	Operator      string
	Value         interface{}
	ValueProvider IValueProvider
}

// OrderClause defines a single order by condition.
type OrderClause struct {
	Field     string
	Direction firestore.Direction
}
