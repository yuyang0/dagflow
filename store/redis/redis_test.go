package redis

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuyang0/dagflow/types"
)

func TestStore_Get(t *testing.T) {
	// Create a mock Redis client
	db, mock := redismock.NewClientMock()
	cfg := &types.RedisConfig{
		Expire: 10,
	}
	store := &Store{
		cli: db,
		cfg: cfg,
	}

	ctx := context.Background()

	t.Run("key exists", func(t *testing.T) {
		key := "_dagflow_testkey"
		value := "testvalue"
		mock.ExpectGet(key).SetVal(value)

		result, err := store.Get(ctx, "testkey")
		require.NoError(t, err)
		assert.Equal(t, []byte(value), result)
	})

	t.Run("key does not exist", func(t *testing.T) {
		key := "_dagflow_nonexistent"
		mock.ExpectGet(key).RedisNil()

		result, err := store.Get(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("redis error", func(t *testing.T) {
		key := "_dagflow_error"
		mock.ExpectGet(key).SetErr(fmt.Errorf("redis error"))

		result, err := store.Get(ctx, "error")
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestStore_Set(t *testing.T) {
	// Create a mock Redis client
	db, mock := redismock.NewClientMock()

	// Define the Redis configuration
	cfg := &types.RedisConfig{
		Expire: 60,
	}

	// Create a new Store instance with the mock client
	store := &Store{
		cli: db,
		cfg: cfg,
	}

	// Define the key and value to set
	key := "test_key"
	value := []byte("test_value")
	redisKey := "_dagflow_" + key

	// Set up the expected Redis command
	mock.ExpectSet(redisKey, value, time.Duration(cfg.Expire)*time.Second).SetVal("OK")

	// Call the Set method
	err := store.Set(context.Background(), key, value)

	// Assert that there were no errors
	assert.NoError(t, err)

	// Ensure that all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}
