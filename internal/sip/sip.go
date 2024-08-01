package sip

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/enums"
	"github.com/artefactual-sdps/preprocessing-sfa/internal/fsutil"
)

type SIP struct {
	Type          enums.SIPType
	Path          string
	ContentPath   string
	MetadataPath  string
	XSDPath       string
	TopLevelPaths []string
}

func NewSIP(path string) (*SIP, error) {
	s := SIP{Path: path}

	if _, err := os.Stat(s.Path); err != nil {
		return nil, fmt.Errorf("NewSIP: %v", err)
	}

	if fsutil.FileExists(filepath.Join(s.Path, "additional")) {
		return s.digitizedAIP(), nil
	}

	f, err := fsutil.FindFilename(s.Path, "Prozess_Digitalisierung_PREMIS.xml")
	if err != nil {
		return nil, fmt.Errorf("NewSIP: %v", err)
	}
	if len(f) > 0 && strings.Contains(s.Path, "Vecteur") {
		return s.digitizedSIP(), nil
	}

	return s.bornDigital(), nil
}

func (s *SIP) digitizedAIP() *SIP {
	s.Type = enums.SIPTypeDigitizedAIP
	s.ContentPath = filepath.Join(s.Path, "content", "content")
	s.MetadataPath = filepath.Join(s.Path, "additional", "UpdatedAreldaMetadata.xml")
	s.XSDPath = filepath.Join(s.Path, "content", "header", "xsd", "arelda.xsd")
	s.TopLevelPaths = []string{
		filepath.Join(s.Path, "content"),
		filepath.Join(s.Path, "additional"),
	}

	return s
}

func (s *SIP) digitizedSIP() *SIP {
	s.bornDigital()
	s.Type = enums.SIPTypeDigitizedSIP

	return s
}

func (s *SIP) bornDigital() *SIP {
	s.Type = enums.SIPTypeBornDigital
	s.ContentPath = filepath.Join(s.Path, "content")
	s.MetadataPath = filepath.Join(s.Path, "header", "metadata.xml")
	s.XSDPath = filepath.Join(s.Path, "header", "xsd", "arelda.xsd")
	s.TopLevelPaths = []string{s.ContentPath, filepath.Join(s.Path, "header")}

	return s
}
