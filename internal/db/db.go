package db

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/log"
	_ "github.com/jackc/pgx/v5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Get pg database connection
func GetDbConnection() (*gorm.DB, error) {
	pgsql, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Error("failed to connect to postgres")
		return nil, err
	}
	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn: pgsql,
	}), &gorm.Config{})
	if err != nil {
		log.Error("failed to get gorm connection", "error", err)
		return nil, err
	}
	return db, nil
}

// Pagination helper
func Paginate(page int, limit int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		offset := (page - 1) * limit
		return db.Offset(offset).Limit(limit)
	}
}

func PaginateRaw(query string, page int, limit int) string {
	query = strings.TrimSuffix(query, ";")
	offset := (page - 1) * limit
	return query + " OFFSET " + fmt.Sprintf("%d", offset) + " LIMIT " + fmt.Sprintf("%d", limit) + ";"
}
