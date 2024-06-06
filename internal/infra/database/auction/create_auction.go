package auction

import (
	"context"
	"fmt"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"sync"
	"time"
)

var lock sync.Mutex

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
}
type AuctionRepository struct {
	Collection *mongo.Collection
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	return &AuctionRepository{
		Collection: database.Collection("auctions"),
	}
}

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {
	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		Timestamp:   auctionEntity.Timestamp.Unix(),
	}
	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)

	go func(ctx context.Context, auctionId string, ar *AuctionRepository) {
		time.Sleep(30 * time.Second)
		closeAuctionIfStillOpen(ctx, auctionId, ar)
	}(ctx, auctionEntity.Id, ar)

	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	return nil
}

func closeAuctionIfStillOpen(ctx context.Context, auctionId string, ar *AuctionRepository) *internal_error.InternalError {
	lock.Lock()
	defer lock.Unlock()

	fmt.Println("Auction with id", auctionId, "is expired")

	filter := bson.M{"_id": auctionId, "status": auction_entity.Active}
	update := bson.M{
		"$set": bson.M{"status": auction_entity.Completed},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.Before)

	var updatedDocument bson.M
	err := ar.Collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedDocument)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return internal_error.NewInternalServerError("No auction found to update")
		}
		logger.Error(fmt.Sprintf("Error trying to update auction by id = %s", auctionId), err)
		return internal_error.NewInternalServerError("Error trying to update auction by id")
	}

	fmt.Println("Auction with id", auctionId, "has been updated to completed")

	return nil
}
