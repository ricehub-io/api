package cache

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const testKey = "connTest"

var rdb *redis.Client

func InitCache(connUrl string) {
	log := zap.L()
	log.Info("Trying to establish a connection with redis cache...")

	opt, err := redis.ParseURL(connUrl)
	if err != nil {
		log.Fatal("Failed to parse redis connection URL", zap.Error(err))
	}

	rdb = redis.NewClient(opt)

	// test the connection
	ctx := context.Background()

	// set test data
	num, err := rand.Int(rand.Reader, big.NewInt(15000))
	if err != nil {
		log.Fatal("Failed to generate random test number", zap.Error(err))
	}
	numStr := num.String()
	err = rdb.Set(ctx, testKey, numStr, 5*time.Minute).Err()
	if err != nil {
		log.Fatal("Failed to set test data in cache", zap.Error(err))
	}

	// retrieve the data
	res, err := rdb.Get(ctx, testKey).Result()
	if err != nil {
		log.Fatal("Failed to retrieve test data from cache", zap.Error(err))
	}
	if res != numStr {
		log.Fatal(
			"Incorrect test data returned from cache",
			zap.String("expected", numStr),
			zap.String("got", res),
		)
	}

	log.Info("Successfully connected to the cache")
}

func CloseCache() {
	if err := rdb.Close(); err != nil {
		zap.L().Fatal("Failed to close cache", zap.Error(err))
	}
}

func increment(key string, expireAfter time.Duration) (int64, error) {
	ctx := context.Background()

	count, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	if count == 1 {
		rdb.Expire(ctx, key, expireAfter)
	}

	return count, err
}

func IncrementRateLimit(clientID string, expireAfter time.Duration) (int64, error) {
	key := fmt.Sprintf("rateLimit:%s", clientID)
	return increment(key, expireAfter)
}

func IncrementPathRateLimit(path string, clientID string, expireAfter time.Duration) (int64, error) {
	key := fmt.Sprintf("pathRateLimit:%s-%s", path, clientID)
	return increment(key, expireAfter)
}
