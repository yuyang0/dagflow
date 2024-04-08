package main

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/hibiken/asynq"
	"github.com/yuyang0/dagflow/flow"
	"github.com/yuyang0/dagflow/service"
	"github.com/yuyang0/dagflow/types"
)

const redisAddr = "127.0.0.1:6379"

func main() {
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
	flowName, nodeName := "f1", "n1"
	createSvcAndFlow(mux, flowName, nodeName)
	// ...register other handlers...

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.Run(mux); err != nil {
			log.Fatalf("could not run server: %v", err)
		}
	}()
	clientSVC := createClientSVC(flowName, nodeName)
	if _, err := clientSVC.Submit(flowName, []byte(`1`)); err != nil {
		log.Fatal("failed to submit task", err)
	}
	wg.Wait()
}

func createSvcAndFlow(mux *asynq.ServeMux, flowName, nodeName string) (*service.Service, *flow.Flow) {
	svc, err := service.New(&types.Config{
		Redis: types.RedisConfig{
			Addr:   redisAddr,
			Expire: 120,
		},
	}, nil)
	if err != nil {
		log.Fatal("failed to create service", err)
	}
	f, err := svc.NewFlow(flowName)
	if err != nil {
		log.Fatal("failed to create flow", err)
	}
	err = f.Node(nodeName, incOp)
	if err != nil {
		log.Fatal("failed to create node", err)
	}
	if err := svc.RegisterFlows(mux, f); err != nil {
		log.Fatalf("failed to register flow: %v", err)
	}
	return svc, f
}

func createClientSVC(flowName, nodeName string) *service.Service {
	svc, _ := createSvcAndFlow(nil, flowName, nodeName)
	return svc
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
