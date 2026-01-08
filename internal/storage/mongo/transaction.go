package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/lugondev/go-carbon/internal/storage"
)

type mongoTransactionRepository struct {
	collection *mongo.Collection
}

func (r *mongoTransactionRepository) Save(ctx context.Context, tx *storage.TransactionModel) error {
	opts := options.Update().SetUpsert(true)
	filter := bson.M{"signature": tx.Signature}
	update := bson.M{"$set": tx}
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (r *mongoTransactionRepository) SaveBatch(ctx context.Context, transactions []*storage.TransactionModel) error {
	if len(transactions) == 0 {
		return nil
	}

	models := make([]mongo.WriteModel, 0, len(transactions))
	for _, tx := range transactions {
		filter := bson.M{"signature": tx.Signature}
		update := bson.M{"$set": tx}
		model := mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update).SetUpsert(true)
		models = append(models, model)
	}

	_, err := r.collection.BulkWrite(ctx, models)
	return err
}

func (r *mongoTransactionRepository) FindBySignature(ctx context.Context, signature string) (*storage.TransactionModel, error) {
	var tx storage.TransactionModel
	err := r.collection.FindOne(ctx, bson.M{"signature": signature}).Decode(&tx)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

func (r *mongoTransactionRepository) FindBySlot(ctx context.Context, slot uint64, limit int, offset int) ([]*storage.TransactionModel, error) {
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset))
	cursor, err := r.collection.Find(ctx, bson.M{"slot": slot}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var transactions []*storage.TransactionModel
	if err := cursor.All(ctx, &transactions); err != nil {
		return nil, err
	}
	return transactions, nil
}

func (r *mongoTransactionRepository) FindByAccountKey(ctx context.Context, accountKey string, limit int, offset int) ([]*storage.TransactionModel, error) {
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset)).SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"account_keys": accountKey}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var transactions []*storage.TransactionModel
	if err := cursor.All(ctx, &transactions); err != nil {
		return nil, err
	}
	return transactions, nil
}

func (r *mongoTransactionRepository) FindRecent(ctx context.Context, limit int) ([]*storage.TransactionModel, error) {
	opts := options.Find().SetLimit(int64(limit)).SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var transactions []*storage.TransactionModel
	if err := cursor.All(ctx, &transactions); err != nil {
		return nil, err
	}
	return transactions, nil
}
