package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/lugondev/go-carbon/internal/storage"
)

type mongoInstructionRepository struct {
	collection *mongo.Collection
}

func (r *mongoInstructionRepository) Save(ctx context.Context, instruction *storage.InstructionModel) error {
	_, err := r.collection.InsertOne(ctx, instruction)
	return err
}

func (r *mongoInstructionRepository) SaveBatch(ctx context.Context, instructions []*storage.InstructionModel) error {
	helper := storage.NewMongoBatchHelper[*storage.InstructionModel](r.collection)
	return helper.InsertMany(ctx, instructions)
}

func (r *mongoInstructionRepository) FindBySignature(ctx context.Context, signature string) ([]*storage.InstructionModel, error) {
	opts := options.Find().SetSort(bson.D{{Key: "instruction_index", Value: 1}})
	cursor, err := r.collection.Find(ctx, bson.M{"signature": signature}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var instructions []*storage.InstructionModel
	if err := cursor.All(ctx, &instructions); err != nil {
		return nil, err
	}
	return instructions, nil
}

func (r *mongoInstructionRepository) FindByProgramID(ctx context.Context, programID string, limit int, offset int) ([]*storage.InstructionModel, error) {
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset)).SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"program_id": programID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var instructions []*storage.InstructionModel
	if err := cursor.All(ctx, &instructions); err != nil {
		return nil, err
	}
	return instructions, nil
}

type mongoEventRepository struct {
	collection *mongo.Collection
}

func (r *mongoEventRepository) Save(ctx context.Context, event *storage.EventModel) error {
	_, err := r.collection.InsertOne(ctx, event)
	return err
}

func (r *mongoEventRepository) SaveBatch(ctx context.Context, events []*storage.EventModel) error {
	helper := storage.NewMongoBatchHelper[*storage.EventModel](r.collection)
	return helper.InsertMany(ctx, events)
}

func (r *mongoEventRepository) FindBySignature(ctx context.Context, signature string) ([]*storage.EventModel, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"signature": signature})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []*storage.EventModel
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (r *mongoEventRepository) FindByProgramID(ctx context.Context, programID string, limit int, offset int) ([]*storage.EventModel, error) {
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset)).SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"program_id": programID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []*storage.EventModel
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (r *mongoEventRepository) FindByEventName(ctx context.Context, eventName string, limit int, offset int) ([]*storage.EventModel, error) {
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset)).SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"event_name": eventName}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []*storage.EventModel
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (r *mongoEventRepository) FindBySlot(ctx context.Context, slot uint64, limit int, offset int) ([]*storage.EventModel, error) {
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset))
	cursor, err := r.collection.Find(ctx, bson.M{"slot": slot}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []*storage.EventModel
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

type mongoTokenAccountRepository struct {
	collection *mongo.Collection
}

func (r *mongoTokenAccountRepository) Save(ctx context.Context, tokenAccount *storage.TokenAccountModel) error {
	opts := options.Update().SetUpsert(true)
	filter := bson.M{"address": tokenAccount.Address}
	update := bson.M{"$set": tokenAccount}
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (r *mongoTokenAccountRepository) SaveBatch(ctx context.Context, tokenAccounts []*storage.TokenAccountModel) error {
	if len(tokenAccounts) == 0 {
		return nil
	}

	models := make([]mongo.WriteModel, 0, len(tokenAccounts))
	for _, ta := range tokenAccounts {
		filter := bson.M{"address": ta.Address}
		update := bson.M{"$set": ta}
		model := mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update).SetUpsert(true)
		models = append(models, model)
	}

	_, err := r.collection.BulkWrite(ctx, models)
	return err
}

func (r *mongoTokenAccountRepository) FindByAddress(ctx context.Context, address string) (*storage.TokenAccountModel, error) {
	var tokenAccount storage.TokenAccountModel
	err := r.collection.FindOne(ctx, bson.M{"address": address}).Decode(&tokenAccount)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &tokenAccount, nil
}

func (r *mongoTokenAccountRepository) FindByOwner(ctx context.Context, owner string, limit int, offset int) ([]*storage.TokenAccountModel, error) {
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset))
	cursor, err := r.collection.Find(ctx, bson.M{"owner": owner}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tokenAccounts []*storage.TokenAccountModel
	if err := cursor.All(ctx, &tokenAccounts); err != nil {
		return nil, err
	}
	return tokenAccounts, nil
}

func (r *mongoTokenAccountRepository) FindByMint(ctx context.Context, mint string, limit int, offset int) ([]*storage.TokenAccountModel, error) {
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset))
	cursor, err := r.collection.Find(ctx, bson.M{"mint": mint}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tokenAccounts []*storage.TokenAccountModel
	if err := cursor.All(ctx, &tokenAccounts); err != nil {
		return nil, err
	}
	return tokenAccounts, nil
}
