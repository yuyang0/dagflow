package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/cockroachdb/errors"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuyang0/dagflow/flow"
	"github.com/yuyang0/dagflow/types"
)

func TestSingleNode(t *testing.T) {
	mockRedis := miniredis.RunT(t)
	redisAddr := mockRedis.Addr()
	svc, err := New(&types.Config{
		Redis: types.RedisConfig{
			Addr:   redisAddr,
			Expire: 120,
		},
		Store: types.StoreConfig{
			Type: "redis",
		},
	}, nil)
	require.NoError(t, err)
	flowName := "f1"
	f, err := svc.NewFlow(flowName)
	require.NoError(t, err)

	err = f.Node("n1", incOp)
	require.NoError(t, err)

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			// Specify how many concurrent workers to use
			Concurrency: 10,
			// Optionally specify multiple queues with different priority.
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			// See the godoc for other configuration options
		},
	)
	mux := asynq.NewServeMux()
	svc.RegisterFlows(mux, f)
	// ...register other handlers...

	go func() {
		if err := srv.Run(mux); err != nil {
			log.Fatalf("could not run server: %v", err)
		}
	}()
	time.Sleep(time.Second)

	cases := []struct {
		input    int
		expected int
		sessID   string
	}{
		{10, 11, ""},
		{20, 21, ""},
		{30, 31, ""},
	}
	for idx, c := range cases {
		sessID, err := svc.Submit(flowName, []byte(fmt.Sprintf(`%d`, c.input)))
		require.NoError(t, err)
		cases[idx].sessID = sessID
	}
	time.Sleep(5 * time.Second)
	for _, c := range cases {
		resMap, err := svc.GetResult(flowName, c.sessID)
		require.NoError(t, err)
		require.Len(t, resMap, 1)
		for k, v := range resMap {
			require.Equal(t, k, "n1")
			var i int
			err := json.Unmarshal(v.Resp, &i)
			require.NoError(t, err)
			assert.Equal(t, i, c.expected)
		}
	}
}

func TestSingleNodeFailure(t *testing.T) {
	mockRedis := miniredis.RunT(t)
	redisAddr := mockRedis.Addr()
	svc, err := New(&types.Config{
		Redis: types.RedisConfig{
			Addr:   redisAddr,
			Expire: 120,
		},
		Store: types.StoreConfig{
			Type: "redis",
		},
	}, nil)
	require.NoError(t, err)
	flowName := "f1"
	f, err := svc.NewFlow(flowName)
	require.NoError(t, err)

	err = f.Node("n1", func(data []byte, option map[string][]string) ([]byte, error) {
		cli := redis.NewClient(&redis.Options{
			Addr: redisAddr, // Redis 服务器地址
		})
		count, err := cli.Incr(context.TODO(), string(data)).Result()
		assert.Nil(t, err)
		fmt.Printf("+++++ Now: %s data: %s count: %d\n", time.Now().String(), string(data), count)
		if count < 3 {
			return data, errors.Newf("error intentionly %s, count: %d", string(data), count)
		}
		return data, nil
	})
	require.NoError(t, err)

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			// Specify how many concurrent workers to use
			Concurrency: 10,
			// Optionally specify multiple queues with different priority.
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			// See the godoc for other configuration options
		},
	)
	mux := asynq.NewServeMux()
	svc.RegisterFlows(mux, f)
	// ...register other handlers...

	go func() {
		if err := srv.Run(mux); err != nil {
			log.Fatalf("could not run server: %v", err)
		}
	}()

	cases := []struct {
		input    int
		expected int
		sessID   string
	}{
		{10, 10, ""},
		{20, 20, ""},
		{30, 30, ""},
	}
	for idx, c := range cases {
		sessID, err := svc.Submit(flowName, []byte(fmt.Sprintf(`%d`, c.input)))
		require.NoError(t, err)
		cases[idx].sessID = sessID
	}
	time.Sleep(120 * time.Second)
	cli := redis.NewClient(&redis.Options{
		Addr: redisAddr, // Redis 服务器地址
	})
	for _, c := range cases {
		resMap, err := svc.GetResult(flowName, c.sessID)
		require.NoError(t, err)
		require.Len(t, resMap, 1)
		for k, v := range resMap {
			require.Equal(t, k, "n1")
			var i int
			err := json.Unmarshal(v.Resp, &i)
			require.NoError(t, err)
			assert.Equal(t, i, c.expected)
			count, err := cli.Get(context.Background(), fmt.Sprintf("%d", c.input)).Result()
			assert.Nil(t, err)
			assert.Equal(t, count, "3")
		}
	}
}

func TestMutipleNodes(t *testing.T) {
	mockRedis := miniredis.RunT(t)
	redisAddr := mockRedis.Addr()
	svc, err := New(&types.Config{
		Redis: types.RedisConfig{
			Addr:   redisAddr,
			Expire: 120,
		},
		Store: types.StoreConfig{
			Type: "redis",
		},
	}, nil)
	require.NoError(t, err)
	flowName := "f2"
	f, err := svc.NewFlow(flowName)
	require.NoError(t, err)
	err = prepareMultipleNodeFlow(f)
	require.NoError(t, err)

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			// Specify how many concurrent workers to use
			Concurrency: 10,
			// Optionally specify multiple queues with different priority.
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			// See the godoc for other configuration options
		},
	)
	mux := asynq.NewServeMux()
	svc.RegisterFlows(mux, f)
	// ...register other handlers...

	go func() {
		if err := srv.Run(mux); err != nil {
			log.Fatalf("could not run server: %v", err)
		}
	}()
	time.Sleep(time.Second)
	expectedFN := func(v int) int {
		return (v + 1 + 1) * (v + 1 - 1) * 2
	}
	cases := []struct {
		input    int
		expected int
		sessID   string
	}{
		{10, expectedFN(10), ""},
		{20, expectedFN(20), ""},
		{30, expectedFN(30), ""},
	}
	for idx, c := range cases {
		sessID, err := svc.Submit(flowName, []byte(fmt.Sprintf(`%d`, c.input)))
		require.NoError(t, err)
		cases[idx].sessID = sessID
	}
	time.Sleep(5 * time.Second)
	for _, c := range cases {
		resMap, err := svc.GetResult(flowName, c.sessID)
		require.NoError(t, err)
		require.Len(t, resMap, 1)
		for k, v := range resMap {
			require.Equal(t, k, "l3n1")
			var i int
			err := json.Unmarshal(v.Resp, &i)
			require.NoError(t, err)
			assert.Equal(t, i, c.expected)
		}
	}
}

func incOp(data []byte, option map[string][]string) ([]byte, error) {
	var i int
	if err := json.Unmarshal(data, &i); err != nil {
		return data, err
	}
	i++
	newData, err := json.Marshal(i)
	return newData, err
}

func decOp(data []byte, option map[string][]string) ([]byte, error) {
	var i int
	if err := json.Unmarshal(data, &i); err != nil {
		return data, err
	}
	i--
	newData, err := json.Marshal(i)
	return newData, err
}

func mulOp(data []byte, option map[string][]string) ([]byte, error) {
	var i int
	if err := json.Unmarshal(data, &i); err != nil {
		return data, err
	}
	i *= 2
	newData, err := json.Marshal(i)
	return newData, err
}

func aggFn(dataMap map[string][]byte) ([]byte, error) {
	var i1, i2 int
	if err := json.Unmarshal(dataMap["l2n1"], &i1); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(dataMap["l2n2"], &i2); err != nil {
		return nil, err
	}
	return json.Marshal(i1 * i2)
}

func prepareMultipleNodeFlow(f *flow.Flow) error {
	if err := f.Node("l1n1", incOp); err != nil {
		return err
	}
	if err := f.Node("l2n1", incOp); err != nil {
		return err
	}
	if err := f.Node("l2n2", decOp); err != nil {
		return err
	}
	if err := f.Node("l3n1", mulOp, flow.WithAggregator(aggFn)); err != nil {
		return err
	}
	if err := f.Edge("l1n1", "l2n1"); err != nil {
		return err
	}
	if err := f.Edge("l1n1", "l2n2"); err != nil {
		return err
	}
	if err := f.Edge("l2n1", "l3n1"); err != nil {
		return err
	}
	if err := f.Edge("l2n2", "l3n1"); err != nil {
		return err
	}
	return nil
}
