package apis_test

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/artefactual-sdps/preprocessing-sfa/internal/apis"
)

func TestCustomMetadataMarshal(t *testing.T) {
	t.Parallel()

	t.Run("marshals APIS metadata", func(t *testing.T) {
		t.Parallel()

		data, err := apis.CustomMetadata{
			ImportTaskID: "task-000001",
			Decision:     "Continue and append",
		}.Marshal()
		assert.NilError(t, err)
		assert.Equal(t, string(data), `{"importTaskId":"task-000001","decision":"Continue and append"}`)
	})

	t.Run("rejects missing task ID", func(t *testing.T) {
		t.Parallel()

		_, err := apis.CustomMetadata{}.Marshal()
		assert.ErrorContains(t, err, "requires import task ID")
	})
}

func TestCustomMetadataUnmarshal(t *testing.T) {
	t.Parallel()

	t.Run("unmarshals APIS metadata", func(t *testing.T) {
		t.Parallel()

		var metadata apis.CustomMetadata
		err := metadata.Unmarshal([]byte(`{"importTaskId":"task-000001","decision":"Continue and append"}`))
		assert.NilError(t, err)
		assert.DeepEqual(t, metadata, apis.CustomMetadata{
			ImportTaskID: "task-000001",
			Decision:     "Continue and append",
		})
	})

	t.Run("rejects missing task ID", func(t *testing.T) {
		t.Parallel()

		var metadata apis.CustomMetadata
		err := metadata.Unmarshal([]byte(`{"decision":"Continue and append"}`))
		assert.ErrorContains(t, err, "requires import task ID")
	})

	t.Run("rejects nil destination", func(t *testing.T) {
		t.Parallel()

		var metadata *apis.CustomMetadata
		err := metadata.Unmarshal([]byte(`{"importTaskId":"task-000001","decision":"Continue and append"}`))
		assert.ErrorContains(t, err, "destination is nil")
	})
}
