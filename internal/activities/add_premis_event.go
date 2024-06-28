package activities

import (
	"context"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/premis"
)

const AddPREMISEventName = "add-premis-event"

type AddPREMISEventParams struct {
	PREMISFilePath string
	Agent          premis.Agent
	Type           string
	Failures       []string
}

type AddPREMISEventResult struct{}

type AddPREMISEventActivity struct{}

func NewAddPREMISEvent() *AddPREMISEventActivity {
	return &AddPREMISEventActivity{}
}

func (md *AddPREMISEventActivity) Execute(
	ctx context.Context,
	params *AddPREMISEventParams,
) (*AddPREMISEventResult, error) {
	doc, err := premis.ParseOrInitialize(params.PREMISFilePath)
	if err != nil {
		return nil, err
	}

	eventSummary, err := premis.NewEventSummary(params.Type, params.Failures)
	if err != nil {
		return nil, err
	}

	err = premis.AppendEventXML(doc, eventSummary, params.Agent)
	if err != nil {
		return nil, err
	}

	err = doc.WriteToFile(params.PREMISFilePath)
	if err != nil {
		return nil, err
	}

	return &AddPREMISEventResult{}, nil
}
