package service

import (
	"context"
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
	cfg    *types.Config
	insp   *asynq.Inspector
}

func New(cfg *types.Config, logger *slog.Logger) (*Service, error) {
	if err := cfg.Refine(); err != nil {
		return nil, err
	}
	cli, insp := prepareAsynqClientAndInspector(&cfg.Redis)
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
		cfg:    cfg,
		insp:   insp,
	}
	return svc, nil
}

func (svc *Service) NewFlow(flowName string) (*flow.Flow, error) {
	flow := flow.New(flowName, svc.stor, svc.cli, svc.logger, svc.cfg, svc.insp)
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

func (svc *Service) RegisterFlowsWithDefinitor(mux *asynq.ServeMux, flows map[string]flow.Definitor) error {
	for name, defHandler := range flows {
		flow, err := svc.NewFlow(name)
		if err != nil {
			return err
		}
		if err := defHandler(context.TODO(), flow); err != nil {
			return err
		}
		flow.Register(mux)
		svc.flows.Set(flow.Name, flow)
	}
	return nil
}
func prepareAsynqClientAndInspector(rCfg *types.RedisConfig) (*asynq.Client, *asynq.Inspector) {
	var (
		rOpt asynq.RedisConnOpt
	)
	if len(rCfg.SentinelAddrs) == 0 {
		rOpt = asynq.RedisClientOpt{
			Addr:     rCfg.Addr,
			DB:       rCfg.DB,
			Username: rCfg.Username,
			Password: rCfg.Password,
		}
	} else {
		rOpt = asynq.RedisFailoverClientOpt{
			SentinelAddrs: rCfg.SentinelAddrs,
			MasterName:    rCfg.MasterName,
			DB:            rCfg.DB,
			Username:      rCfg.Username,
			Password:      rCfg.Password,
		}
	}
	cli := asynq.NewClient(rOpt)
	insp := asynq.NewInspector(rOpt)
	return cli, insp
}
