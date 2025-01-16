package persistence

type Config struct {
	Driver  string
	DSN     string
	Migrate bool
}
