package mongodb

import (
	"context"
	"errors"
	"fmt"

	"github.com/namanv3/go-utils/helpers"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoClient[T any] interface {
	Insert(object T, ctx context.Context) (bool, error)
	Replace(query bson.M, object T, upsert bool, ctx context.Context) (bool, error)
	InsertMany(objects []T, ctx context.Context) error
	Find(query bson.M, ctx context.Context) (*T, error)
	List(query bson.M, ctx context.Context) ([]T, error)
	Aggregate(pipeline bson.A, ctx context.Context) ([]T, error)
	ListWithSortOptions(query bson.M, sortOptions bson.D, ctx context.Context) ([]T, error)
	Update(query bson.M, updateFields bson.M, upsert bool, ctx context.Context) (*T, bool, error)
	UpdateWithOptions(query bson.M, updateFields bson.M, updateOptions options.FindOneAndUpdateOptions, ctx context.Context) (*T, bool, error)
	UpdateManyWithOptions(query bson.M, updateFields bson.M, updateOptions options.UpdateOptions, ctx context.Context) (bool, error)
	UpdateMany(query bson.M, updateFields bson.M, upsert bool, ctx context.Context) (int64, bool, error)
	Delete(query bson.M, ctx context.Context) (deletedCount int64, err error)
}

type DefaultMongoClient[T any] struct {
	client     *mongo.Client
	db         string
	collection string
}

func NewMongoClient[T any](connection *mongo.Client, db, collection string) MongoClient[T] {
	return DefaultMongoClient[T]{
		client:     connection,
		db:         db,
		collection: collection,
	}
}

func (c DefaultMongoClient[T]) Insert(object T, ctx context.Context) (bool, error) {
	db := c.client.Database(c.db)
	collection := db.Collection(c.collection)
	_, err := collection.InsertOne(ctx, object)
	if err != nil {
		helpers.LogError(err, "unexpected error when inserting object into mongo", map[string]any{"objectToInsert": object, "collection": c.collection}, ctx)
		return false, errors.New("unexpected error when inserting object into mongo")
	}
	return true, nil
}

func (c DefaultMongoClient[T]) Replace(query bson.M, object T, upsert bool, ctx context.Context) (bool, error) {
	db := c.client.Database(c.db)
	collection := db.Collection(c.collection)
	result, err := collection.ReplaceOne(ctx, query, object, options.Replace().SetUpsert(upsert))
	if err != nil {
		helpers.LogError(err, "unexpected error when replacing object in mongo", map[string]any{"query": query, "replacement": object, "collection": c.collection}, ctx)
		return false, errors.New("unexpected error when replacing object in mongo")
	}
	return result.MatchedCount > 0, nil
}

func (c DefaultMongoClient[T]) InsertMany(objects []T, ctx context.Context) error {
	db := c.client.Database(c.db)
	collection := db.Collection(c.collection)
	interfaceList := []any{}
	for _, obj := range objects {
		interfaceList = append(interfaceList, obj)
	}
	result, err := collection.InsertMany(ctx, interfaceList)
	if err != nil {
		helpers.LogError(err, "unexpected error when inserting object into mongo", map[string]any{"objectsToInsert": objects, "collection": c.collection}, ctx)
		return errors.New("unexpected error when inserting object into mongo")
	} else if len(result.InsertedIDs) != len(objects) {
		return fmt.Errorf("only able to insert %d out of %d objects", len(result.InsertedIDs), len(objects))
	}
	return nil
}

func (c DefaultMongoClient[T]) Find(query bson.M, ctx context.Context) (*T, error) {
	db := c.client.Database(c.db)
	collection := db.Collection(c.collection)

	result := collection.FindOne(ctx, query)
	err := result.Err()
	if err == mongo.ErrNoDocuments {
		helpers.LogInfo("no documents found for given query", map[string]any{"query": query, "collection": c.collection}, ctx)
		return nil, nil
	} else if err != nil {
		helpers.LogError(err, "unexpected error when finding object in mongo", map[string]any{"query": query, "collection": c.collection}, ctx)
		return nil, errors.New("unexpected error when finding object in mongo")
	}

	var object T
	if err := result.Decode(&object); err != nil {
		helpers.LogError(err, "unexpected error when decoding object found in mongo", map[string]any{"query": query, "collection": c.collection}, ctx)
		return nil, errors.New("unexpected error when decoding object found in mongo")
	}
	return &object, nil
}

func (c DefaultMongoClient[T]) List(query bson.M, ctx context.Context) ([]T, error) {
	return c.ListWithSortOptions(query, bson.D{}, ctx)
}
func (c DefaultMongoClient[T]) ListWithSortOptions(query bson.M, sortOptions bson.D, ctx context.Context) ([]T, error) {
	db := c.client.Database(c.db)
	collection := db.Collection(c.collection)

	opts := options.Find().SetSort(sortOptions)
	cursor, err := collection.Find(ctx, query, opts)
	if err != nil || cursor.Err() != nil {
		helpers.LogError(err, "unexpected error when finding objects in mongo", map[string]any{"query": query, "sortOptions": sortOptions, "collection": c.collection}, ctx)
		return nil, errors.New("unexpected error when finding objects in mongo")
	}
	defer cursor.Close(ctx)

	elements := []T{}
	for cursor.Next(ctx) {
		var element T
		if err := cursor.Decode(&element); err != nil {
			helpers.LogError(err, "unexpected error when decoding object found in mongo", map[string]any{"query": query, "current": cursor.Current, "collection": c.collection}, ctx)
			return nil, errors.New("unexpected error when decoding object found in mongo")
		}
		elements = append(elements, element)
	}
	return elements, nil
}

func (c DefaultMongoClient[T]) Aggregate(pipeline bson.A, ctx context.Context) ([]T, error) {
	db := c.client.Database(c.db)
	collection := db.Collection(c.collection)

	cursor, err := collection.Aggregate(context.TODO(), pipeline)
	if err != nil || cursor.Err() != nil {
		helpers.LogError(err, "unexpected error when finding objects in mongo using pipeline", map[string]any{"pipeline": pipeline, "collection": c.collection}, ctx)
		return nil, errors.New("unexpected error when finding objects in mongo using pipeline")
	}
	defer cursor.Close(ctx)
	elements := []T{}
	for cursor.Next(ctx) {
		var element T
		if err := cursor.Decode(&element); err != nil {
			helpers.LogError(err, "unexpected error when decoding object found in mongo using pipeline", map[string]any{"pipeline": pipeline, "collection": c.collection}, ctx)
			return nil, errors.New("unexpected error when decoding object found in mongo using pipeline")
		}
		elements = append(elements, element)
	}
	return elements, nil
}

func (c DefaultMongoClient[T]) Update(query bson.M, updateFields bson.M, upsert bool, ctx context.Context) (*T, bool, error) {
	db := c.client.Database(c.db)
	collection := db.Collection(c.collection)

	result := collection.FindOneAndUpdate(ctx, query, updateFields, options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(upsert))

	err := result.Err()
	if err == mongo.ErrNoDocuments {
		helpers.LogError(err, "no documents found for given query", map[string]any{"query": query, "update": updateFields, "upsert": upsert, "collection": c.collection}, ctx)
		return nil, false, nil
	} else if err != nil {
		helpers.LogError(err, "unexpected error when updating object in mongo", map[string]any{"query": query, "update": updateFields, "upsert": upsert, "collection": c.collection}, ctx)
		return nil, false, errors.New("unexpected error when updating object in mongo")
	}

	var object T
	if err := result.Decode(&object); err != nil {
		helpers.LogError(err, "unexpected error when decoding object updated in mongo", map[string]any{"query": query, "update": updateFields, "upsert": upsert, "collection": c.collection}, ctx)
		return nil, true, errors.New("unexpected error when decoding object updated in mongo")
	}
	return &object, true, nil
}

func (c DefaultMongoClient[T]) UpdateWithOptions(query bson.M, updateFields bson.M, updateOptions options.FindOneAndUpdateOptions, ctx context.Context) (*T, bool, error) {
	db := c.client.Database(c.db)
	collection := db.Collection(c.collection)

	result := collection.FindOneAndUpdate(ctx, query, updateFields, &updateOptions)

	err := result.Err()
	if err == mongo.ErrNoDocuments {
		helpers.LogError(err, "no documents found for given query", map[string]any{"query": query, "update": updateFields, "collection": c.collection}, ctx)
		return nil, false, nil
	} else if err != nil {
		helpers.LogError(err, "unexpected error when updating object in mongo", map[string]any{"query": query, "update": updateFields, "collection": c.collection}, ctx)
		return nil, false, errors.New("unexpected error when updating object in mongo")
	}

	var object T
	if err := result.Decode(&object); err != nil {
		helpers.LogError(err, "unexpected error when decoding object updated in mongo", map[string]any{"query": query, "update": updateFields, "collection": c.collection}, ctx)
		return nil, true, errors.New("unexpected error when decoding object updated in mongo")
	}
	return &object, true, nil
}

func (c DefaultMongoClient[T]) UpdateManyWithOptions(query bson.M, updateFields bson.M, updateOptions options.UpdateOptions, ctx context.Context) (bool, error) {
	db := c.client.Database(c.db)
	collection := db.Collection(c.collection)

	result, err := collection.UpdateMany(ctx, query, updateFields, &updateOptions)

	if err == mongo.ErrNoDocuments || result.MatchedCount == 0 {
		helpers.LogError(err, "no documents found for given query", map[string]any{"query": query, "update": updateFields, "collection": c.collection}, ctx)
		return false, nil
	} else if err != nil {
		helpers.LogError(err, "unexpected error when updating object in mongo", map[string]any{"query": query, "update": updateFields, "collection": c.collection}, ctx)
		return false, errors.New("unexpected error when updating object in mongo")
	}
	return true, nil
}

func (c DefaultMongoClient[T]) UpdateMany(query bson.M, updateFields bson.M, upsert bool, ctx context.Context) (int64, bool, error) {
	db := c.client.Database(c.db)
	collection := db.Collection(c.collection)

	result, err := collection.UpdateMany(ctx, query, updateFields, options.Update().SetUpsert(upsert))
	if err == mongo.ErrNoDocuments {
		helpers.LogError(err, "no documents found for given query", map[string]any{"query": query, "update": updateFields, "upsert": upsert, "collection": c.collection}, ctx)
		return 0, false, nil
	} else if err != nil {
		helpers.LogError(err, "unexpected error when updating object in mongo", map[string]any{"query": query, "update": updateFields, "upsert": upsert, "collection": c.collection}, ctx)
		return 0, false, errors.New("unexpected error when updating object in mongo")
	}
	return result.ModifiedCount, true, nil
}

func (c DefaultMongoClient[T]) Delete(query bson.M, ctx context.Context) (deletedCount int64, err error) {
	db := c.client.Database(c.db)
	collection := db.Collection(c.collection)

	result, err := collection.DeleteMany(ctx, query)
	if err != nil {
		helpers.LogError(err, "unexpected error when deleting objects in mongo", map[string]any{"query": query, "collection": c.collection}, ctx)
		return 0, errors.New("unexpected error when deleting objects in mongo")
	}
	return result.DeletedCount, nil
}
