package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/lugondev/go-carbon/internal/storage"
)

type mongoAccountRepository struct {
	collection *mongo.Collection
}

func (r *mongoAccountRepository) Save(ctx context.Context, account *storage.AccountModel) error {
	opts := options.Update().SetUpsert(true)
	filter := bson.M{"pubkey": account.Pubkey}
	update := bson.M{"$set": account}
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (r *mongoAccountRepository) SaveBatch(ctx context.Context, accounts []*storage.AccountModel) error {
	if len(accounts) == 0 {
		return nil
	}

	models := make([]mongo.WriteModel, 0, len(accounts))
	for _, account := range accounts {
		filter := bson.M{"pubkey": account.Pubkey}
		update := bson.M{"$set": account}
		model := mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update).SetUpsert(true)
		models = append(models, model)
	}

	_, err := r.collection.BulkWrite(ctx, models)
	return err
}

func (r *mongoAccountRepository) FindByPubkey(ctx context.Context, pubkey string) (*storage.AccountModel, error) {
	var account storage.AccountModel
	err := r.collection.FindOne(ctx, bson.M{"pubkey": pubkey}).Decode(&account)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (r *mongoAccountRepository) FindByOwner(ctx context.Context, owner string, limit int, offset int) ([]*storage.AccountModel, error) {
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset)).SetSort(bson.D{{Key: "slot", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"owner": owner}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var accounts []*storage.AccountModel
	if err := cursor.All(ctx, &accounts); err != nil {
		return nil, err
	}
	return accounts, nil
}

func (r *mongoAccountRepository) FindBySlot(ctx context.Context, slot uint64, limit int, offset int) ([]*storage.AccountModel, error) {
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset))
	cursor, err := r.collection.Find(ctx, bson.M{"slot": slot}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var accounts []*storage.AccountModel
	if err := cursor.All(ctx, &accounts); err != nil {
		return nil, err
	}
	return accounts, nil
}

func (r *mongoAccountRepository) Delete(ctx context.Context, pubkey string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"pubkey": pubkey})
	return err
}
