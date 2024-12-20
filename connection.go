package fireorm

import (
	"cloud.google.com/go/firestore"
	"fmt"
)

type IConnection interface {
	Validate() error
	GetClient() *firestore.Client
	GetTransaction() *firestore.Transaction
	HasTransaction() bool
	HasClient() bool
	Close() error
	SetTransaction(tx *firestore.Transaction) IConnection
	SetClient(client *firestore.Client) IConnection
}

type Connection struct {
	client      *firestore.Client
	transaction *firestore.Transaction
}

func NewConnection(client *firestore.Client, transaction ...*firestore.Transaction) *Connection {
	c := &Connection{client: client}
	if len(transaction) > 0 && transaction[0] != nil {
		c.transaction = transaction[0]
	}
	return c
}

func (c *Connection) Validate() error {
	if !c.HasClient() {
		return fmt.Errorf("firestore client is required")
	}
	return nil
}

func (c *Connection) GetClient() *firestore.Client {
	return c.client
}

func (c *Connection) GetTransaction() *firestore.Transaction {
	return c.transaction
}

func (c *Connection) HasTransaction() bool {
	return c.transaction != nil
}

func (c *Connection) HasClient() bool {
	return c.client != nil
}

func (c *Connection) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

func (c *Connection) SetTransaction(tx *firestore.Transaction) IConnection {
	c.transaction = tx
	return c
}

func (c *Connection) SetClient(client *firestore.Client) IConnection {
	c.client = client
	return c
}
