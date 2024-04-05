package main

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/hibiken/asynq"
	"github.com/yuyang0/dagflow/service"
	"github.com/yuyang0/dagflow/types"
)

const redisAddr = "127.0.0.1:6379"

func main() {
	svc, err := service.New(&types.Config{
		Store: types.StoreConfig{
			Type: "redis",
			Redis: types.RedisConfig{
				Addr:   redisAddr,
				Expire: 120,
			},
		},
	}, nil)
	if err != nil {
		log.Fatal("failed to create service", err)
	}
	f, err := svc.NewFlow("f1")
	if err != nil {
		log.Fatal("failed to create flow", err)
	}
	err = f.Node("n1", incOp)
	if err != nil {
		log.Fatal("failed to create node", err)
	}

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

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.Run(mux); err != nil {
			log.Fatalf("could not run server: %v", err)
		}
	}()
	svc.Submit("f1", []byte(`1`))
	wg.Wait()
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
