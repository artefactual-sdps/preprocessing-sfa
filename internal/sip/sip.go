package sip

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
)

type SIP struct {
	Type         enums.SIPType
	Path         string
	ContentPath  string
	MetadataPath string
	XSDPath      string
}

func NewSIP(path string) (SIP, error) {
	s := SIP{Path: path}

	if _, err := os.Stat(s.Path); err != nil {
		return s, fmt.Errorf("NewSIP: %v", err)
	}

	if _, err := os.Stat(filepath.Join(s.Path, "additional")); err != nil {
		if !os.IsNotExist(err) {
			return s, fmt.Errorf("NewSIP: %v", err)
		}
		s.Type = enums.SIPTypeVecteurSIP
		s.ContentPath = filepath.Join(s.Path, "content")
		s.MetadataPath = filepath.Join(s.Path, "header", "metadata.xml")
		s.XSDPath = filepath.Join(s.Path, "header", "xsd", "arelda.xsd")
	} else {
		s.Type = enums.SIPTypeVecteurAIP
		s.ContentPath = filepath.Join(s.Path, "content", "content")
		s.MetadataPath = filepath.Join(s.Path, "additional", "UpdatedAreldaMetadata.xml")
		s.XSDPath = filepath.Join(s.Path, "content", "header", "xsd", "arelda.xsd")
	}

	return s, nil
}
