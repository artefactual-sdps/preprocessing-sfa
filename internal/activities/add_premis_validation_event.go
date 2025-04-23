package activities

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/beevik/etree"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/fformat"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/fvalidate"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/premis"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

const AddPREMISValidationEventName = "add-premis-validation-event"

type AddPREMISValidationEventParams struct {
	SIP            sip.SIP
	PREMISFilePath string
	Agent          premis.Agent
	Summary        premis.EventSummary
}

type AddPREMISValidationEventResult struct{}

type AddPREMISValidationEventActivity struct {
	validator fvalidate.Validator
}

func NewAddPREMISValidationEvent(validator fvalidate.Validator) *AddPREMISValidationEventActivity {
	return &AddPREMISValidationEventActivity{
		validator: validator,
	}
}

func (md *AddPREMISValidationEventActivity) Execute(
	ctx context.Context,
	params *AddPREMISValidationEventParams,
) (*AddPREMISValidationEventResult, error) {
	// Load or initialize PREMIS XML.
	doc, err := premis.ParseOrInitialize(params.PREMISFilePath)
	if err != nil {
		return nil, err
	}

	PREMISEl := doc.FindElement("/premis:premis")
	if PREMISEl == nil {
		return nil, fmt.Errorf("no root premis element found in document")
	}

	// Determine formats of SIP files.
	fileformats, err := fformat.IdentifyFormats(ctx, fformat.NewSiegfriedEmbed(), params.SIP)
	if err != nil {
		return nil, fmt.Errorf("identifyFormats: %v", err)
	}

	// Determine which files should have been checked by the validator.
	allowedIds := md.validator.FormatIDs()

	for path, f := range fileformats {
		if slices.Contains(allowedIds, f.ID) {
			// Determine subpath.
			subpath, err := filepath.Rel(params.SIP.ContentPath, path)
			if err != nil {
				return nil, err
			}

			// Find PREMIS object element using original name.
			originalName := premis.OriginalNameForSubpath(params.SIP, subpath)

			objectOriginalNameEl := doc.FindElement(
				fmt.Sprintf("/premis:premis/premis:object/premis:originalName[text()='%s']", originalName),
			)

			if objectOriginalNameEl == nil {
				return nil, fmt.Errorf("element not found")
			}

			objectEl := objectOriginalNameEl.Parent()

			// Append PREMIS event linked to from PREMIS object element.
			var objectEls []*etree.Element
			objectEls = append(objectEls, objectEl)
			premis.AppendEventXMLForObjects(PREMISEl, params.Summary, params.Agent, objectEls)
		}
	}

	// Write PREMIS.
	err = doc.WriteToFile(params.PREMISFilePath)
	if err != nil {
		return nil, err
	}

	return &AddPREMISValidationEventResult{}, nil
}
