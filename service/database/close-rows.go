package database

import (
	"database/sql"
	"log"
)

// closeRows closes SQL rows and reports any closing error
func closeRows(rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		log.Printf("database: failed to close SQL rows: %v", err)
	}
}
