package fireorm

import (
	"cloud.google.com/go/firestore"
	"context"
)

type FirestoreClient interface {
	Doc(path string) FirestoreDocumentRef
	Close() error
}

type FirestoreDocumentRef interface {
	Get(ctx context.Context) (*firestore.DocumentSnapshot, error)
}

type FirestoreClientWrapper struct {
	client *firestore.Client
}

func (f *FirestoreClientWrapper) Doc(path string) FirestoreDocumentRef {
	return &FirestoreDocumentRefWrapper{
		doc: f.client.Doc(path),
	}
}

func (f *FirestoreClientWrapper) Close() error {
	return f.client.Close()
}

type FirestoreDocumentRefWrapper struct {
	doc *firestore.DocumentRef
}

func (f *FirestoreDocumentRefWrapper) Get(ctx context.Context) (*firestore.DocumentSnapshot, error) {
	return f.doc.Get(ctx)
}
