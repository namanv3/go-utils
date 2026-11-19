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
	GetCollectionName() string
	Insert(object T, ctx context.Context) (bool, error)
	Replace(query bson.M, object T, upsert bool, ctx context.Context) (bool, error)
	InsertMany(objects []T, ctx context.Context) error
	Find(query bson.M, ctx context.Context) (*T, error)
	FindWithOptions(query bson.M, findOptions options.FindOneOptions, ctx context.Context) (*T, error)
	List(query bson.M, ctx context.Context) ([]T, error)
	Aggregate(pipeline bson.A, ctx context.Context) ([]T, error)
	AggregateWithCount(pipeline bson.A, limit, offset int64, ctx context.Context) ([]T, int64, error)
	ListWithSortOptions(query bson.M, sortOptions bson.D, ctx context.Context) ([]T, error)
	Update(query bson.M, updateFields bson.M, upsert bool, ctx context.Context) (*T, bool, error)
	UpdateWithOptions(query bson.M, updateFields bson.M, updateOptions options.FindOneAndUpdateOptions, ctx context.Context) (*T, bool, error)
	UpdateManyWithOptions(query bson.M, updateFields bson.M, updateOptions options.UpdateOptions, ctx context.Context) (bool, error)
	UpdateMany(query bson.M, updateFields bson.M, upsert bool, ctx context.Context) (int64, bool, error)
	Delete(query bson.M, ctx context.Context) (deletedCount int64, err error)
	EnsureIndexes(indexes []mongo.IndexModel, ctx context.Context) error
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

// GetCollectionName returns the collection name configured for this client.
func (c DefaultMongoClient[T]) GetCollectionName() string {
	return c.collection
}

func (c DefaultMongoClient[T]) Insert(object T, ctx context.Context) (bool, error) {
	db := c.client.Database(c.db)
	collection := db.Collection(c.collection)
	_, err := collection.InsertOne(ctx, object)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return false, err
		}
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
		if mongo.IsDuplicateKeyError(err) {
			return false, err
		}
		helpers.LogError(err, "unexpected error when replacing object in mongo", map[string]any{"query": query, "replacement": object, "collection": c.collection}, ctx)
		return false, errors.New("unexpected error when replacing object in mongo")
	}
	return result.ModifiedCount > 0 || result.UpsertedCount > 0, nil
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
		if mongo.IsDuplicateKeyError(err) {
			return err
		}
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

func (c DefaultMongoClient[T]) FindWithOptions(query bson.M, findOptions options.FindOneOptions, ctx context.Context) (*T, error) {
	collection := c.client.Database(c.db).Collection(c.collection)
	var object T
	if err := collection.FindOne(ctx, query, &findOptions).Decode(&object); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("unable to find a document in collection %q in database %q: %w", c.collection, c.db, err)
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
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, err
		}
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
	if err := cursor.Err(); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, err
		}
		helpers.LogError(err, "unexpected error when finding objects in mongo using pipeline", map[string]any{"pipeline": pipeline, "collection": c.collection}, ctx)
		return nil, errors.New("unexpected error when finding objects in mongo using pipeline")
	}
	return elements, nil
}

// AggregateWithCount applies pipeline before counting its results and paginating
// them. The pipeline should contain filtering and sorting, without pagination.
func (c DefaultMongoClient[T]) AggregateWithCount(pipeline bson.A, limit, offset int64, ctx context.Context) ([]T, int64, error) {
	stages := make(bson.A, 0, len(pipeline)+1)
	stages = append(stages, pipeline...)
	stages = append(stages, bson.M{"$facet": bson.M{
		"items": bson.A{
			bson.M{"$skip": offset},
			bson.M{"$limit": limit},
		},
		"count": bson.A{bson.M{"$count": "total"}},
	}})

	collection := c.client.Database(c.db).Collection(c.collection)
	cursor, err := collection.Aggregate(ctx, stages)
	if err != nil {
		return nil, 0, fmt.Errorf("unable to aggregate documents with a total count from collection %q in database %q: %w", c.collection, c.db, err)
	}
	defer cursor.Close(ctx)

	var results []struct {
		Items []T `bson:"items"`
		Count []struct {
			Total int64 `bson:"total"`
		} `bson:"count"`
	}
	if err := cursor.All(ctx, &results); err != nil {
		return nil, 0, fmt.Errorf("unable to read aggregation results with a total count from collection %q in database %q: %w", c.collection, c.db, err)
	}
	if len(results) == 0 {
		return []T{}, 0, nil
	}
	var total int64
	if len(results[0].Count) > 0 {
		total = results[0].Count[0].Total
	}
	return results[0].Items, total, nil
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
		if mongo.IsDuplicateKeyError(err) {
			return nil, false, err
		}
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
		if mongo.IsDuplicateKeyError(err) {
			return nil, false, err
		}
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

	if err == mongo.ErrNoDocuments || (err == nil && result.MatchedCount == 0) {
		helpers.LogError(err, "no documents found for given query", map[string]any{"query": query, "update": updateFields, "collection": c.collection}, ctx)
		return false, nil
	} else if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return false, err
		}
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
		if mongo.IsDuplicateKeyError(err) {
			return 0, false, err
		}
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

func (c DefaultMongoClient[T]) EnsureIndexes(indexes []mongo.IndexModel, ctx context.Context) error {
	collection := c.client.Database(c.db).Collection(c.collection)
	if _, err := collection.Indexes().CreateMany(ctx, indexes); err != nil {
		return fmt.Errorf("unable to create indexes for collection %q in database %q: %w", c.collection, c.db, err)
	}
	return nil
}
