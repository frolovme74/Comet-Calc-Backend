package main

import (
	"log"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/frolovme74/Comet-Calc-Backend/internal/dsn"
	"github.com/frolovme74/Comet-Calc-Backend/internal/models"
)

func main() {
	_ = godotenv.Load()

	db, err := gorm.Open(postgres.Open(dsn.FromEnv()), &gorm.Config{})
	if err != nil {
		log.Fatalf("подключение к БД: %v", err)
	}

	if err := db.AutoMigrate(&models.User{}, &models.Comet{}, &models.CometLike{}); err != nil {
		log.Fatalf("миграция: %v", err)
	}
	log.Println("Миграция выполнена: users, comets, comet_likes")
}
