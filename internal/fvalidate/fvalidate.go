package fvalidate

// Validator provides an interface for validating a file's format.
type Validator interface {
	// FormatIDs lists the format IDs that the validator can validate.
	FormatIDs() []string

	// Name of the validator.
	Name() string

	// Validate validates the file at path.
	Validate(path string) (string, error)

	// Returns the version of a validator.
	Version() (string, error)
}
