package indexer

import (
	"errors"

	"gorm.io/gorm"
)

type KVStore struct {
	ID    string `gorm:"primaryKey"`
	Value []byte
}

// Postgres Storage implementation
type PgStorage struct {
	db *gorm.DB
}

func (s *PgStorage) Get(id []byte) []byte {
	var val KVStore
	val.ID = string(id)
	tx := s.db.First(&val)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil
	}
	return val.Value
}

func (s *PgStorage) Has(id []byte) bool {
	var val KVStore
	val.ID = string(id)
	tx := s.db.First(&val)

	return tx.Error == nil
}

func (s *PgStorage) Set(key []byte, value []byte) error {
	val := KVStore{
		ID:    string(key),
		Value: value,
	}
	err := s.db.Save(&val)
	return err.Error
}

func (s *PgStorage) Scan(prefix []byte) [][]byte {
	var results [][]byte
	s.db.Model(&KVStore{}).Where("id LIKE ?", string(prefix)+"%").Select("value").Find(&results)

	return results
}

func NewPgStorage(db *gorm.DB) *PgStorage {
	return &PgStorage{
		db,
	}
}
