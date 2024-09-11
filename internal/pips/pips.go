package pips

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/sip"
)

type PIP struct {
	// Type is the type of SIP (DigitizedAIP, DigitizedSIP, BornDigital).
	Type enums.SIPType

	// Path is the filepath of the PIP directory.
	Path string

	// ManifestPath is the filepath of the SIP manifest —
	// "UpdatedAreldaMetadata.xml" for digitized AIPs, "metadata.xml" for all
	// other SIP types.
	ManifestPath string
}

func New(path string, t enums.SIPType) PIP {
	p := PIP{Path: path, Type: t}
	if p.Type == enums.SIPTypeDigitizedAIP {
		p.ManifestPath = filepath.Join(path, "metadata", "UpdatedAreldaMetadata.xml")
	} else {
		p.ManifestPath = filepath.Join(path, "objects", p.Name(), "header", "metadata.xml")
	}

	return p
}

func NewFromSIP(sip sip.SIP) PIP {
	return New(sip.Path, sip.Type)
}

func (p PIP) Name() string {
	return filepath.Base(p.Path)
}

func (p PIP) ConvertSIPPath(path string) string {
	switch {
	case filepath.Base(path) == "Prozess_Digitalisierung_PREMIS.xml":
		parent := filepath.Base(filepath.Dir(path))
		return filepath.Join(
			"metadata",
			fmt.Sprintf("Prozess_Digitalisierung_PREMIS_%s.xml", parent),
		)
	case filepath.Base(path) == "metadata.xml":
		return filepath.Join("objects", p.Name(), "header", "metadata.xml")
	case filepath.Base(path) == "UpdatedAreldaMetadata.xml":
		return filepath.Join("metadata", "UpdatedAreldaMetadata.xml")
	case strings.HasPrefix(path, "content"):
		return filepath.Join("objects", p.Name(), path)
	default:
		return ""
	}
}
