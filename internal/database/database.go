package database

import (
	"log"

	"time"

	"github.com/lmaobamar/cygnet-backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(dsn string) *gorm.DB {
	loadStart := time.Now()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		log.Fatalf("db: cant connect: %v", err)
	}
	log.Printf("db in %v", time.Since(loadStart))
	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("db: migrate failed : %v", err)
	} else {
		log.Printf("db.AutoMigrate in %v", time.Since(loadStart))
	}

	return db
}
