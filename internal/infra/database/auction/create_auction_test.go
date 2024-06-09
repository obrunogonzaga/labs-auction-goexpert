package auction

import (
	"context"
	"errors"
	"fmt"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"sync"
	"testing"
	"time"
)

func setupMongoContainer(ctx context.Context) (mongoURI string, cleanup func()) {
	req := testcontainers.ContainerRequest{
		Image:        "mongo:latest",
		ExposedPorts: []string{"27017/tcp"},
		WaitingFor:   wait.ForLog("Waiting for connections").WithStartupTimeout(120 * time.Second),
	}
	mongoC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Fatal(err)
	}

	ip, err := mongoC.Host(ctx)
	if err != nil {
		log.Fatal(err)
	}

	port, err := mongoC.MappedPort(ctx, "27017")
	if err != nil {
		log.Fatal(err)
	}

	mongoURI = fmt.Sprintf("mongodb://%s:%s", ip, port.Port())

	return mongoURI, func() {
		mongoC.Terminate(ctx)
	}
}

func TestCloseAuctionIfStillOpen(t *testing.T) {
	ctx := context.Background()

	mongoURI, cleanup := setupMongoContainer(ctx)
	defer cleanup()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("auctions")
	ar := &AuctionRepository{
		Collection: db.Collection("auctions"),
	}

	_, err = ar.Collection.InsertOne(ctx, bson.M{"_id": "123", "status": auction_entity.Active})
	if err != nil {
		t.Fatalf("Failed to insert initial document: %v", err)
	}

	err = closeAuctionIfStillOpen(ctx, "123", ar)
	if err.(*internal_error.InternalError) == nil {
		fmt.Println("Function executed successfully, no error.")
	} else {
		t.Errorf("Function failed with non-nil error: %v", err)
	}

	var result bson.M
	if err := ar.Collection.FindOne(ctx, bson.M{"_id": "123"}).Decode(&result); err != nil {
		t.Errorf("Failed to fetch updated document: %v", err)
	}

	statusValue, ok := result["status"].(int32)
	if !ok {
		t.Errorf("Failed to assert status as int32")
		return
	}

	if statusValue != int32(auction_entity.Completed) {
		t.Errorf("Auction status was not updated properly. Got %v, want %v", statusValue, auction_entity.Completed)
	}
}

func TestCreateAuction(t *testing.T) {
	ctx := context.Background()

	mongoURI, cleanup := setupMongoContainer(ctx)
	defer cleanup()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("auctions")
	ar := &AuctionRepository{
		Collection: db.Collection("auctions"),
	}

	auction := &auction_entity.Auction{
		Id:          "abc123",
		ProductName: "Vintage Toy",
		Category:    "Toys",
		Description: "A classic vintage toy from the 1980s",
		Condition:   auction_entity.ProductCondition(auction_entity.Used),
		Status:      auction_entity.Active,
		Timestamp:   time.Now(),
	}

	err = ar.CreateAuction(ctx, auction)
	if err.(*internal_error.InternalError) == nil {
		fmt.Println("Function executed successfully, no error.")
	} else {
		t.Errorf("Function failed with non-nil error: %v", err)
	}

	var result bson.M
	err = ar.Collection.FindOne(ctx, bson.M{"_id": "abc123"}).Decode(&result)
	if err != nil {
		var internalErr *internal_error.InternalError
		if errors.As(err, &internalErr) {
			t.Errorf("Failed to fetch auction document: %v", internalErr)
		} else {
			t.Errorf("Failed to fetch auction document: %v", err)
		}
	}

	if result["status"] != int32(auction_entity.Active) {
		t.Errorf("Auction status is incorrect. Got %v, want %v", result["status"], auction_entity.Active)
	}

	time.Sleep(35 * time.Second)

	if err := ar.Collection.FindOne(ctx, bson.M{"_id": "abc123"}).Decode(&result); err != nil {
		t.Errorf("Failed to fetch updated auction document: %v", err)
	}

	if result["status"] != int32(auction_entity.Completed) {
		t.Errorf("Auction status was not updated to completed as expected. Got %v, want %v", result["status"], auction_entity.Completed)
	}
}

func TestConcurrentCloseAuction(t *testing.T) {
	ctx := context.Background()

	mongoURI, cleanup := setupMongoContainer(ctx)
	defer cleanup()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("auctions")
	ar := &AuctionRepository{
		Collection: db.Collection("auctions"),
	}

	var wg sync.WaitGroup
	const numGoroutines = 10

	// Cria vários leilões
	for i := 0; i < numGoroutines; i++ {
		auctionID := fmt.Sprintf("auction_%d", i)
		_, err = ar.Collection.InsertOne(ctx, bson.M{"_id": auctionID, "status": auction_entity.Active})
		if err != nil {
			t.Fatalf("Failed to insert initial document: %v", err)
		}
	}

	// Executa múltiplas goroutines para testar concorrência
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(auctionID string) {
			defer wg.Done()
			if err := closeAuctionIfStillOpen(ctx, auctionID, ar); err != nil {
				t.Errorf("Function failed with error: %v", err)
			}
		}(fmt.Sprintf("auction_%d", i))
	}

	wg.Wait()

	// Verifica resultados
	for i := 0; i < numGoroutines; i++ {
		auctionID := fmt.Sprintf("auction_%d", i)
		var result bson.M
		if err := ar.Collection.FindOne(ctx, bson.M{"_id": auctionID}).Decode(&result); err != nil {
			t.Errorf("Failed to fetch updated document: %v", err)
		}

		statusValue, ok := result["status"].(int32)
		if !ok {
			t.Errorf("Failed to assert status as int32")
			continue
		}

		if statusValue != int32(auction_entity.Completed) {
			t.Errorf("Auction status was not updated properly for auction %s. Got %v, want %v", auctionID, statusValue, auction_entity.Completed)
		}
	}
}
