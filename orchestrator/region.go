package orchestrator

import (
	"strings"

	aw "github.com/deanishe/awgo"
	"github.com/dineshgowda24/alfred-gcp-workflow/gcloud"
	"github.com/dineshgowda24/alfred-gcp-workflow/workflow/log"
)

var _ Handler = (*RegionHandler)(nil)

type RegionHandler struct{}

func (r *RegionHandler) Handle(ctx *Context) error {
	log.Debug("region handler started")

	regions := gcloud.GetAllRegions()
	for _, region := range regions {
		r.addRegion(ctx, &region)
	}

	r.send(ctx)
	log.Debug("region handler completed")
	return nil
}

func (r *RegionHandler) addRegion(ctx *Context, region *gcloud.Region) {
	newQuery := r.buildConfigAutocomplete(ctx, region)
	ctx.Workflow.NewItem(region.Name).
		Subtitle(region.Location).
		Autocomplete(newQuery).
		Arg(newQuery).
		Match(region.Name + " " + region.Location).
		Icon(aw.IconNetwork).
		Valid(true)
}

func (r *RegionHandler) buildConfigAutocomplete(ctx *Context, region *gcloud.Region) string {
	query := ctx.Args.Query
	partial := ctx.ParsedQuery.PartialRegionQuery
	return strings.Replace(query, "$"+partial, "$"+region.Name, 1)
}

func (r *RegionHandler) send(ctx *Context) {
	wf := ctx.Workflow
	if strings.TrimSpace(ctx.ParsedQuery.PartialRegionQuery) != "" {
		wf.Filter(ctx.ParsedQuery.PartialRegionQuery)
	}

	if wf.IsEmpty() {
		emptyResultItem(wf, "No matching regions found")
	}

	wf.SendFeedback()
}
