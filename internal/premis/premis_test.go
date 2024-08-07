package premis_test

import (
	"errors"
	"testing"

	"github.com/beevik/etree"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/premis"
)

const premisObjectAddContent = `<?xml version="1.0" encoding="UTF-8"?>
<premis:premis xmlns:premis="http://www.loc.gov/premis/v3" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.loc.gov/premis/v3 https://www.loc.gov/standards/premis/premis.xsd" version="3.0">
  <premis:object xsi:type="premis:file">
    <premis:objectIdentifier>
      <premis:objectIdentifierType>uuid</premis:objectIdentifierType>
      <premis:objectIdentifierValue>c74a85b7-919b-409e-8209-9c7ebe0e7945</premis:objectIdentifierValue>
    </premis:objectIdentifier>
    <premis:objectCharacteristics>
      <premis:format>
        <premis:formatDesignation>
          <premis:formatName/>
        </premis:formatDesignation>
      </premis:format>
    </premis:objectCharacteristics>
    <premis:originalName>data/objects/cat.jpg</premis:originalName>
  </premis:object>
</premis:premis>
`

const premisEventAddContent = `<?xml version="1.0" encoding="UTF-8"?>
<premis:premis xmlns:premis="http://www.loc.gov/premis/v3" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.loc.gov/premis/v3 https://www.loc.gov/standards/premis/premis.xsd" version="3.0">
  <premis:event>
    <premis:eventIdentifier>
      <premis:eventIdentifierType>UUID</premis:eventIdentifierType>
      <premis:eventIdentifierValue/>
    </premis:eventIdentifier>
    <premis:eventType>validation</premis:eventType>
    <premis:eventDateTime/>
    <premis:eventDetailInformation>
      <premis:eventDetail>event detail</premis:eventDetail>
    </premis:eventDetailInformation>
    <premis:eventOutcomeInformation>
      <premis:eventOutcome>valid</premis:eventOutcome>
    </premis:eventOutcomeInformation>
    <premis:linkingAgentIdentifier>
      <premis:linkingAgentIdentifierType valueURI="http://id.loc.gov/vocabulary/identifiers/local">url</premis:linkingAgentIdentifierType>
      <premis:linkingAgentIdentifierValue>https://github.com/artefactual-sdps/preprocessing-sfa</premis:linkingAgentIdentifierValue>
    </premis:linkingAgentIdentifier>
  </premis:event>
</premis:premis>
`

const premisObjectAndEventAddContent = `<?xml version="1.0" encoding="UTF-8"?>
<premis:premis xmlns:premis="http://www.loc.gov/premis/v3" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.loc.gov/premis/v3 https://www.loc.gov/standards/premis/premis.xsd" version="3.0">
  <premis:object xsi:type="premis:file">
    <premis:objectIdentifier>
      <premis:objectIdentifierType>uuid</premis:objectIdentifierType>
      <premis:objectIdentifierValue>c74a85b7-919b-409e-8209-9c7ebe0e7945</premis:objectIdentifierValue>
    </premis:objectIdentifier>
    <premis:objectCharacteristics>
      <premis:format>
        <premis:formatDesignation>
          <premis:formatName/>
        </premis:formatDesignation>
      </premis:format>
    </premis:objectCharacteristics>
    <premis:originalName>data/objects/cat.jpg</premis:originalName>
    <premis:linkingEventIdentifier>
      <premis:linkingEventIdentifierType>UUID</premis:linkingEventIdentifierType>
      <premis:linkingEventIdentifierValue/>
    </premis:linkingEventIdentifier>
  </premis:object>
  <premis:event>
    <premis:eventIdentifier>
      <premis:eventIdentifierType>UUID</premis:eventIdentifierType>
      <premis:eventIdentifierValue/>
    </premis:eventIdentifier>
    <premis:eventType>validation</premis:eventType>
    <premis:eventDateTime/>
    <premis:eventDetailInformation>
      <premis:eventDetail>name=&quot;Validate SIP metadata&quot;</premis:eventDetail>
    </premis:eventDetailInformation>
    <premis:eventOutcomeInformation>
      <premis:eventOutcome>invalid</premis:eventOutcome>
      <premis:eventOutcomeDetail>
        <premis:eventOutcomeDetailNote>Metadata validation successful</premis:eventOutcomeDetailNote>
      </premis:eventOutcomeDetail>
    </premis:eventOutcomeInformation>
    <premis:linkingAgentIdentifier>
      <premis:linkingAgentIdentifierType valueURI="http://id.loc.gov/vocabulary/identifiers/local">url</premis:linkingAgentIdentifierType>
      <premis:linkingAgentIdentifierValue>https://github.com/artefactual-sdps/preprocessing-sfa</premis:linkingAgentIdentifierValue>
    </premis:linkingAgentIdentifier>
  </premis:event>
</premis:premis>
`

const premisAgentAddContent = `<?xml version="1.0" encoding="UTF-8"?>
<premis:premis xmlns:premis="http://www.loc.gov/premis/v3" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.loc.gov/premis/v3 https://www.loc.gov/standards/premis/premis.xsd" version="3.0">
  <premis:agent>
    <premis:agentIdentifier>
      <premis:agentIdentifierType valueURI="http://id.loc.gov/vocabulary/identifiers/local">url</premis:agentIdentifierType>
      <premis:agentIdentifierValue>https://github.com/artefactual-sdps/preprocessing-sfa</premis:agentIdentifierValue>
    </premis:agentIdentifier>
    <premis:agentName>Enduro</premis:agentName>
    <premis:agentType>software</premis:agentType>
  </premis:agent>
</premis:premis>
`

func TestAppendPREMISObjectXML(t *testing.T) {
	// Test with PREMIS object.
	uuid := "c74a85b7-919b-409e-8209-9c7ebe0e7945"
	originalName := "data/objects/cat.jpg"

	doc, err := premis.NewDoc()
	assert.NilError(t, err)

	err = premis.AppendObjectXML(doc, premis.Object{
		IdType:       "uuid",
		IdValue:      uuid,
		OriginalName: originalName,
	})
	assert.NilError(t, err)

	// Get resulting XML string.
	doc.Indent(2)
	xml, err := doc.WriteToString()
	assert.NilError(t, err)

	// Compare XML to constant.
	assert.Equal(t, xml, premisObjectAddContent)
}

func TestAppendPREMISEventXML(t *testing.T) {
	// Test with PREMIS event.
	doc, err := premis.NewDoc()
	assert.NilError(t, err)

	err = premis.AppendEventXML(doc, premis.EventSummary{
		Type:    "validation",
		Detail:  "event detail",
		Outcome: "valid",
	}, premis.AgentDefault())
	assert.NilError(t, err)

	// Check length then blank the event identifier value.
	idValueEl := doc.FindElement("/premis:premis/premis:event/premis:eventIdentifier/premis:eventIdentifierValue")
	assert.Assert(t, idValueEl != nil)
	assert.Assert(t, len(idValueEl.Text()) == 36)
	idValueEl.SetText("")

	// Blank the event datetime.
	err = blankElementText(doc, "/premis:premis/premis:event/premis:eventDateTime")
	assert.NilError(t, err)

	// Get resulting XML string.
	doc.Indent(2)
	xml, err := doc.WriteToString()
	assert.NilError(t, err)

	// Compare XML to constant.
	assert.Equal(t, xml, premisEventAddContent)
}

func TestAppendPREMISAgentXML(t *testing.T) {
	// Test with PREMIS agent.
	doc, err := premis.NewDoc()
	assert.NilError(t, err)

	err = premis.AppendAgentXML(doc, premis.AgentDefault())
	assert.NilError(t, err)

	// Get resulting XML string.
	doc.Indent(2)
	xml, err := doc.WriteToString()
	assert.NilError(t, err)

	// Compare XML to constant.
	assert.Equal(t, xml, premisAgentAddContent)

	// Try to add another PREMIS agent to existing XML document.
	doc = etree.NewDocument()
	doc.ReadFromString(xml)
	doc.Indent(2)

	err = premis.AppendAgentXML(doc, premis.AgentDefault())
	assert.NilError(t, err)

	// Get resulting XML string.
	doc.Indent(2)
	xml, err = doc.WriteToString()
	assert.NilError(t, err)

	// Compare XML to constant to make sure a duplicate agent wasn't added.
	assert.Equal(t, xml, premisAgentAddContent)
}

func TestFilesWithinDirectory(t *testing.T) {
	contentPath := fs.NewDir(t, "",
		fs.WithDir("content",
			fs.WithDir("content",
				fs.WithDir("d_0000001",
					fs.WithFile("00000001.jp2", ""),
					fs.WithFile("00000001_PREMIS.xml", ""),
					fs.WithFile("Prozess_Digitalisierung_PREMIS.xml", ""),
				),
			),
		),
	).Path()

	expectedFiles := []string{
		"content/content/d_0000001/00000001.jp2",
		"content/content/d_0000001/00000001_PREMIS.xml",
		"content/content/d_0000001/Prozess_Digitalisierung_PREMIS.xml",
	}

	files, err := premis.FilesWithinDirectory(contentPath)
	assert.NilError(t, err)

	assert.DeepEqual(t, files, expectedFiles)
}

func TestOriginalNameForSubpath(t *testing.T) {
	// Check for correct adjustment of file paths in PREMIS.
	originalName := premis.OriginalNameForSubpath(
		"content/content/d_0000001/00000001.jp2",
	)

	assert.Equal(t, originalName,
		"data/objects/content/content/d_0000001/00000001.jp2")

	// Check for special handling of this specific file's path in PREMIS.
	originalName = premis.OriginalNameForSubpath(
		"content/content/d_0000001/Prozess_Digitalisierung_PREMIS.xml",
	)

	assert.Equal(t, originalName,
		"data/metadata/Prozess_Digitalisierung_PREMIS_d_0000001.xml")
}

func TestAppendPREMISEventAndLinkToObject(t *testing.T) {
	// Define PREMIS event with failure.
	var failures []string
	failures = append(failures, "some failure")
	outcome := premis.EventOutcomeForFailures(failures)

	// Add PREMIS event to XML document.
	eventSummary, err := premis.NewEventSummary(
		"validation",
		"name=\"Validate SIP metadata\"",
		outcome,
		"Metadata validation successful",
	)
	assert.NilError(t, err)

	doc := etree.NewDocument()
	err = doc.ReadFromString(premisObjectAddContent)
	assert.NilError(t, err)

	originalName := premis.OriginalNameForSubpath("cat.jpg")

	err = premis.AppendEventAndLinkToObject(doc, eventSummary, premis.AgentDefault(), originalName)
	assert.NilError(t, err)

	// Blank text for random/time elements.
	err = blankElementText(
		doc,
		"/premis:premis/premis:object/premis:linkingEventIdentifier/premis:linkingEventIdentifierValue",
	)
	assert.NilError(t, err)

	err = blankElementText(doc, "/premis:premis/premis:event/premis:eventIdentifier/premis:eventIdentifierValue")
	assert.NilError(t, err)

	err = blankElementText(doc, "/premis:premis/premis:event/premis:eventDateTime")
	assert.NilError(t, err)

	// Check modifed XML output.
	doc.Indent(2)
	eventXml, err := doc.WriteToString()
	assert.NilError(t, err)
	assert.Equal(t, eventXml, premisObjectAndEventAddContent)
}

func blankElementText(doc *etree.Document, xpath string) error {
	el := doc.FindElement(xpath)
	if el == nil {
		return errors.New("element not found")
	}

	el.SetText("")

	return nil
}
