package orchestrator

import (
	"embed"
	"errors"

	aw "github.com/deanishe/awgo"
	"github.com/dineshgowda24/alfred-gcp-workflow/gcloud"
	"github.com/dineshgowda24/alfred-gcp-workflow/parser"
	"github.com/dineshgowda24/alfred-gcp-workflow/searchers"
	"github.com/dineshgowda24/alfred-gcp-workflow/services"
	"github.com/dineshgowda24/alfred-gcp-workflow/workflow/arg"
	"github.com/dineshgowda24/alfred-gcp-workflow/workflow/log"
)

var ErrNoActiveConfig = errors.New("no active gcloud config found")

type Handler interface {
	Handle(ctx *Context) error
}

type Orchestrator struct {
	ctx               *Context
	preflight         *PreFlight
	intentHandler     Handler
	configHandler     Handler
	regionHandler     Handler
	homeHandler       Handler
	serviceHandler    Handler
	subServiceHandler Handler
	fallback          Handler
	errorHandler      Handler
	servicesFS        embed.FS
}

func DefaultOrchestrator(servicesFS embed.FS) *Orchestrator {
	return NewOrchestrator(
		&ConfigHandler{},
		&IntentHandler{},
		&HomeHandler{},
		&RegionHandler{},
		&ServiceHandler{},
		&SubServiceHandler{},
		&FallbackHandler{},
		&ErrorHandler{},
		&PreFlight{},
		servicesFS,
	)
}

func NewOrchestrator(
	config, intent, home, region, service, subService, fallback, err Handler,
	preflight *PreFlight, servicesFS embed.FS,
) *Orchestrator {
	return &Orchestrator{
		preflight:         preflight,
		intentHandler:     intent,
		configHandler:     config,
		regionHandler:     region,
		homeHandler:       home,
		serviceHandler:    service,
		subServiceHandler: subService,
		fallback:          fallback,
		errorHandler:      err,
		servicesFS:        servicesFS,
	}
}

func (o *Orchestrator) Run(wf *aw.Workflow, args *arg.SearchArgs) {
	log.Debug("running orchestrator with query:", args.Query)

	o.buildCtx(wf, args)
	if o.ctx.Err != nil {
		o.handleErr()
		return
	}

	log.Debug("build context completed")

	switch {
	case o.ctx.IsIntentQuery():
		o.ctx.Err = o.intentHandler.Handle(o.ctx)
	case (o.ctx.IsConfigQuery() && !o.ctx.IsConfigActive()):
		o.ctx.Err = o.configHandler.Handle(o.ctx)
	case o.ctx.IsRegionQuery() && !o.ctx.IsRegionActive():
		o.ctx.Err = o.regionHandler.Handle(o.ctx)
	case o.ctx.IsHomeQuery():
		o.ctx.Err = o.homeHandler.Handle(o.ctx)
	case o.ctx.IsSubServiceQuery():
		o.ctx.Err = o.subServiceHandler.Handle(o.ctx)
	case o.ctx.IsServiceQuery():
		o.ctx.Err = o.serviceHandler.Handle(o.ctx)
	default:
		o.ctx.Err = o.fallback.Handle(o.ctx)
	}

	o.handleErr()
}

func (o *Orchestrator) buildCtx(wf *aw.Workflow, args *arg.SearchArgs) {
	ctx := &Context{Workflow: wf, Args: args}
	o.ctx = ctx
	servicesList, err := services.Load(o.servicesFS)
	if err != nil {
		ctx.Err = err
		return
	}

	ctx.Services = servicesList
	ctx.SearchRegistry = searchers.GetDefaultRegistry()
	ctx.ParsedQuery = parser.Parse(args, servicesList)

	if err := o.preflightCheck(); err != nil {
		ctx.Err = err
		return
	}

	if ctx.ParsedQuery.Config != nil {
		ctx.ActiveConfig = ctx.ParsedQuery.Config
	} else if !ctx.IsIntentQuery() {
		config := gcloud.GetActiveConfig()
		if config == nil {
			ctx.Err = ErrNoActiveConfig
			return
		}

		ctx.ActiveConfig = config
	}

	if ctx.ParsedQuery.Region != nil {
		ctx.ActiveConfig.Region = ctx.ParsedQuery.Region
	}
}

func (o *Orchestrator) handleErr() {
	if o.ctx.Err != nil {
		o.errorHandler.Handle(o.ctx)
	}
}

func (o *Orchestrator) preflightCheck() error {
	err := o.preflight.Check(o.ctx.Workflow, o.ctx.ParsedQuery)
	if err != nil {
		log.Error("preflight check failed:", err)
		return err
	}
	return nil
}
