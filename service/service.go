package service

import (
	"log/slog"
	"os"

	"github.com/alphadose/haxmap"
	"github.com/hibiken/asynq"
	"github.com/yuyang0/dagflow/flow"
	"github.com/yuyang0/dagflow/store"
	"github.com/yuyang0/dagflow/store/factory"
	"github.com/yuyang0/dagflow/types"
)

type Service struct {
	flows  *haxmap.Map[string, *flow.Flow]
	cli    *asynq.Client
	stor   store.Store
	logger *slog.Logger
}

func New(cfg *types.Config, cli *asynq.Client, logger *slog.Logger) (*Service, error) {
	stor, err := factory.NewStore(&cfg.Store)
	if err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	svc := &Service{
		flows:  haxmap.New[string, *flow.Flow](),
		stor:   stor,
		cli:    cli,
		logger: logger,
	}
	return svc, nil
}

func (svc *Service) NewFlow(flowName string) (*flow.Flow, error) {
	flow := flow.New(flowName, svc.stor, svc.cli, svc.logger)
	return flow, nil
}

// submit a flow task
func (svc *Service) Submit(flowName string, body []byte) (string, error) {
	flow, ok := svc.flows.Get(flowName)
	if !ok {
		return "", types.ErrNoFlow
	}
	return flow.Submit(body)
}

func (svc *Service) GetResult(flowName string, sessID string) (map[string]*flow.ExecResult, error) {
	flow, ok := svc.flows.Get(flowName)
	if !ok {
		return nil, types.ErrNoFlow
	}
	return flow.GetResult(sessID)
}

func (svc *Service) RegisterFlows(mux *asynq.ServeMux, flows ...*flow.Flow) error {
	for _, flow := range flows {
		flow.Register(mux)
		svc.flows.Set(flow.Name, flow)
	}
	return nil
}
