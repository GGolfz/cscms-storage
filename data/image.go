package data

import (
	"errors"
	"github.com/thetkpark/cscms-temp-storage/data/model"
	"gorm.io/gorm"
)

type ImageDataStore interface {
	Create(image *model.Image) error
	FindByUserID(userID uint) (*[]model.Image, error)
	FindByID(imageID uint) (*model.Image, error)
	DeleteByID(imageId uint) error
}

type GormImageDataStore struct {
	db *gorm.DB
}

func NewGormImageDataStore(db *gorm.DB) (*GormImageDataStore, error) {
	if err := db.AutoMigrate(&model.Image{}); err != nil {
		return nil, err
	}
	return &GormImageDataStore{
		db: db,
	}, nil
}

func (g *GormImageDataStore) Create(image *model.Image) error {
	tx := g.db.Create(image)
	return tx.Error
}

func (g *GormImageDataStore) DeleteByID(imageId uint) error {
	tx := g.db.Delete(&model.Image{}, imageId)
	return tx.Error
}

func (g *GormImageDataStore) FindByID(imageID uint) (*model.Image, error) {
	var image model.Image
	tx := g.db.Where(&model.User{ID: imageID}).First(&image)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &image, tx.Error
}

func (g *GormImageDataStore) FindByUserID(userID uint) (*[]model.Image, error) {
	var images []model.Image
	tx := g.db.Where(&model.Image{UserID: userID}).Find(&images)
	return &images, tx.Error
}
