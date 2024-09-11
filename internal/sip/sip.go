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
	// Type is the type of SIP (DigitizedAIP, DigitizedSIP, BornDigital).
	Type enums.SIPType

	// Path is the filepath of the SIP directory.
	Path string

	// ContentPath is the filepath of the "content" directory.
	ContentPath string

	// ManifestPath is the filepath of the SIP manifest —
	// "UpdatedAreldaMetadata.xml" for digitized AIPs, "metadata.xml" for all
	// other SIP types.
	ManifestPath string

	// MetadataPath is the path of the "metadata.xml" file.
	MetadataPath string

	// UpdatedAreldaMDPath is the filepath of the "UpdatedAreldaMetadata.xml"
	// file for digitized AIPs — it is empty for all other SIP types.
	UpdatedAreldaMDPath string

	// XSDPath is the filepath of the "arelda.xsd" file.
	XSDPath string

	// TopLevelPaths is a list of all the top level SIP directories.
	TopLevelPaths []string
}

func New(path string) (SIP, error) {
	s := SIP{}

	if _, err := os.Stat(path); err != nil {
		return s, fmt.Errorf("SIP: New: %v", err)
	}
	s.Path = path

	if fsutil.FileExists(filepath.Join(s.Path, "additional")) {
		return s.digitizedAIP(), nil
	}

	f, err := fsutil.FindFilename(s.Path, "Prozess_Digitalisierung_PREMIS.xml")
	if err != nil {
		return s, fmt.Errorf("SIP: New: %v", err)
	}
	if len(f) > 0 && strings.Contains(strings.ToLower(s.Path), "vecteur") {
		return s.digitizedSIP(), nil
	}

	return s.bornDigital(), nil
}

func (s SIP) digitizedAIP() SIP {
	s.Type = enums.SIPTypeDigitizedAIP
	s.ContentPath = filepath.Join(s.Path, "content", "content")
	s.MetadataPath = filepath.Join(s.Path, "content", "header", "old", "SIP", "metadata.xml")
	s.UpdatedAreldaMDPath = filepath.Join(s.Path, "additional", "UpdatedAreldaMetadata.xml")
	s.ManifestPath = s.UpdatedAreldaMDPath
	s.XSDPath = filepath.Join(s.Path, "content", "header", "xsd", "arelda.xsd")
	s.TopLevelPaths = []string{
		filepath.Join(s.Path, "content"),
		filepath.Join(s.Path, "additional"),
	}

	return s
}

func (s SIP) digitizedSIP() SIP {
	s = s.bornDigital()
	s.Type = enums.SIPTypeDigitizedSIP

	return s
}

func (s SIP) bornDigital() SIP {
	s.Type = enums.SIPTypeBornDigital
	s.ContentPath = filepath.Join(s.Path, "content")
	s.MetadataPath = filepath.Join(s.Path, "header", "metadata.xml")
	s.ManifestPath = s.MetadataPath
	s.XSDPath = filepath.Join(s.Path, "header", "xsd", "arelda.xsd")
	s.TopLevelPaths = []string{
		filepath.Join(s.Path, "content"),
		filepath.Join(s.Path, "header"),
	}

	return s
}

func (s SIP) Name() string {
	return filepath.Base(s.Path)
}
