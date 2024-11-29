package handlers

import (
	"context"
	"fmt"

	"github.com/namanv3/go-utils/helpers"
	"github.com/namanv3/go-utils/mongodb"
	"go.mongodb.org/mongo-driver/bson"
)

type CRUDHandler[T any] interface {
	Create(element T, ctx context.Context) error
	GetOne(query bson.M, ctx context.Context) (*T, error)
	GetAll(ctx context.Context) ([]T, error)
	List(query bson.M, ctx context.Context) ([]T, error)
}

type DefaultCRUDHandler[T any] struct {
	mongoClient mongodb.MongoClient[T]
	elementName string
}

func NewCRUDHandler[T any](mongoClient mongodb.MongoClient[T], elementName string) CRUDHandler[T] {
	return DefaultCRUDHandler[T]{
		mongoClient: mongoClient,
		elementName: elementName,
	}
}

func (h DefaultCRUDHandler[T]) Create(element T, ctx context.Context) error {
	inserted, err := h.mongoClient.Insert(element, ctx)
	if err != nil || !inserted {
		helpers.LogError(err, fmt.Sprintf("unexpected error when inserting %s", h.elementName), map[string]any{h.elementName: element}, ctx)
		return fmt.Errorf("unexpected error when inserting %s", h.elementName)
	}
	return nil
}

func (h DefaultCRUDHandler[T]) GetOne(query bson.M, ctx context.Context) (*T, error) {
	element, err := h.mongoClient.Find(query, ctx)
	if err != nil {
		helpers.LogError(err, fmt.Sprintf("unexpected error when fetching list of %ss", h.elementName), map[string]any{"query": query}, ctx)
		return nil, fmt.Errorf("unexpected error when fetching list of %ss", h.elementName)
	}
	return element, nil
}

func (h DefaultCRUDHandler[T]) GetAll(ctx context.Context) ([]T, error) {
	return h.List(bson.M{}, ctx)
}

func (h DefaultCRUDHandler[T]) List(query bson.M, ctx context.Context) ([]T, error) {
	elements, err := h.mongoClient.List(query, ctx)
	if err != nil {
		helpers.LogError(err, fmt.Sprintf("unexpected error when fetching list of %ss", h.elementName), map[string]any{"query": query}, ctx)
		return nil, fmt.Errorf("unexpected error when fetching list of %ss", h.elementName)
	}
	return elements, nil
}
