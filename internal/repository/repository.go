package repository

import (
	"errors"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	ErrNotFound    = errors.New("запись не найдена")
	ErrDraftExists = errors.New("у пользователя уже есть черновик")
)

type Repository struct {
	db *gorm.DB
}

func New(dsn string) (*Repository, error) {
	config := &gorm.Config{TranslateError: true}
	if os.Getenv("DB_LOG") == "1" {
		config.Logger = logger.Default.LogMode(logger.Info)
	}
	db, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		return nil, err
	}
	return &Repository{db: db}, nil
}

func notFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
