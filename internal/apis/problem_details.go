package apis

import "github.com/artefactual-sdps/preprocessing-sfa/internal/apis/gen"

func problemDetail(detail gen.OptNilString) string {
	if value, ok := detail.Get(); ok && value != "" {
		return value
	}

	return "no additional details"
}
