package apis

import (
	"encoding/json"
	"fmt"
)

const CustomMetadataKey = "apis"

type CustomMetadata struct {
	ImportTaskID string `json:"importTaskId"`
	Decision     string `json:"decision"`
}

func (m CustomMetadata) Marshal() ([]byte, error) {
	if m.ImportTaskID == "" {
		return nil, fmt.Errorf("APIS custom metadata requires import task ID")
	}

	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal APIS custom metadata: %w", err)
	}

	return data, nil
}

func (m *CustomMetadata) Unmarshal(data []byte) error {
	if m == nil {
		return fmt.Errorf("APIS custom metadata destination is nil")
	}
	if err := json.Unmarshal(data, m); err != nil {
		return fmt.Errorf("unmarshal APIS custom metadata: %w", err)
	}
	if m.ImportTaskID == "" {
		return fmt.Errorf("APIS custom metadata requires import task ID")
	}

	return nil
}
