package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	"github.com/yuyang0/dagflow/flow"
	"github.com/yuyang0/dagflow/service"
	"github.com/yuyang0/dagflow/types"
)

const redisAddr = "127.0.0.1:6379"

func main() {

	mux := asynq.NewServeMux()
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
	flowName := "f2"
	createSvc(mux, flowName)
	// ...register other handlers...

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.Run(mux); err != nil {
			log.Fatalf("could not run server: %v", err)
		}
	}()
	clientSVC := createClientSVC(flowName)
	intialV := 10
	expectV := (10 + 1 + 1) * (10 + 1 - 1) * 2
	sessID, err := clientSVC.Submit("f2", []byte(fmt.Sprintf(`%d`, intialV)))
	if err != nil {
		log.Fatal("failed to submit task", err)
	}
	time.Sleep(15 * time.Second)
	resMap, err := clientSVC.GetResult("f2", sessID)
	if err != nil {
		log.Fatal("failed to get result", err)
	}

	for k, v := range resMap {
		var i int
		json.Unmarshal(v.Resp, &i)
		log.Printf("+++++ %s, actual: %d, expect: %d\n", k, i, expectV)
	}
	wg.Wait()
}

func createSvc(mux *asynq.ServeMux, flowName string) *service.Service {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	svc, err := service.New(&types.Config{
		Redis: types.RedisConfig{
			Addr:   redisAddr,
			Expire: 120,
		},
	}, logger)
	if err != nil {
		log.Fatal("failed to create service", err)
	}
	if err := svc.RegisterFlowsWithDefinitor(mux, map[string]flow.Definitor{
		flowName: prepareFlow,
	}); err != nil {
		log.Fatal("failed to prepare flow", err)
	}
	return svc
}

func createClientSVC(flowName string) *service.Service {
	svc := createSvc(nil, flowName)
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

func prepareFlow(_ context.Context, f *flow.Flow) error {
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
