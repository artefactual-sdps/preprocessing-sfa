package persistence

import (
	"database/sql"
	"fmt"
	"runtime"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

// Open a database driver.
func Open(driver, ds string) (db *sql.DB, err error) {
	switch driver {
	case "mysql":
		db, err = openMySQL(ds)
	case "sqlite3":
		db, err = openSQLite(ds)
	default:
		return nil, fmt.Errorf("database driver %q not supported", driver)
	}

	return db, err
}

// openMySQL opens the MySQL database driver.
func openMySQL(ds string) (*sql.DB, error) {
	config, err := mysql.ParseDSN(ds)
	if err != nil {
		return nil, fmt.Errorf("error parsing dsn: %w (%s)", err, ds)
	}
	config.Collation = "utf8mb4_unicode_ci"
	config.Loc = time.UTC
	config.ParseTime = true
	config.MultiStatements = true
	config.Params = map[string]string{
		"time_zone": "UTC",
	}

	conn, err := mysql.NewConnector(config)
	if err != nil {
		return nil, fmt.Errorf("error creating connector: %w", err)
	}

	sqlDB := sql.OpenDB(conn)
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return sqlDB, nil
}

// openSQLite opens the SQLite database driver.
func openSQLite(ds string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", ds)
	if err != nil {
		return nil, err
	}

	conns := runtime.NumCPU()
	db.SetMaxOpenConns(conns)
	db.SetMaxIdleConns(conns)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	pragmas := []string{
		"journa_mode=WAL",
		"synchronous=OFF",
		"foreign_keys=ON",
		"tempo_store=MEMORY",
		"busy_timeout=1000", // Used with "_txlock=immediate" or "BEGIN IMMEDIATE".
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec("PRAGMA " + pragma); err != nil {
			return nil, err
		}
	}

	return db, nil
}
