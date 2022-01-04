package data

import (
	"github.com/bxcodec/faker/v3"
	"github.com/thetkpark/cscms-temp-storage/data/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"math/rand"
	"os"
	"time"
)

const SqlitePath = "test.db"

func createTestGormDB() (*gorm.DB, error) {
	gormLogger := logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{IgnoreRecordNotFoundError: true})
	return gorm.Open(sqlite.Open(SqlitePath), &gorm.Config{Logger: gormLogger})
}

func destroyTestGormDB() error {
	return os.Remove(SqlitePath)
}

func createTestUser(provider string) *model.User {
	return &model.User{
		ID:        uint(rand.Uint32()),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Email:     faker.Email(),
		Username:  faker.Username(),
		Provider:  provider,
		AvatarURL: faker.URL(),
		Files:     nil,
		Images:    nil,
		APIKey:    faker.UUIDDigit(),
	}
}

func createTestFile(userID uint, expired bool) *model.File {
	file := &model.File{
		ID:        faker.Password(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiredAt: time.Now().Add(time.Hour),
		Token:     faker.Password(),
		Nonce:     faker.UUIDDigit(),
		Filename:  faker.Username(),
		FileSize:  uint64(rand.Uint32()),
		Visited:   uint(rand.Uint32()),
		FileType:  faker.Currency(),
		Encrypted: rand.Int() > rand.Int(),
		UserID:    userID,
	}
	if expired {
		file.ExpiredAt = time.Now().Add(-1 * time.Hour)
	}
	return file
}

func createTestImage(userID uint) *model.Image {
	return &model.Image{
		ID:               uint(rand.Uint32()),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		OriginalFilename: faker.Username(),
		FileSize:         uint64(rand.Uint32()),
		FilePath:         faker.Password() + ".png",
		UserID:           userID,
		DeletedAt:        gorm.DeletedAt{},
	}
}
