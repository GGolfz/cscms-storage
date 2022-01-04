package data

import (
	"errors"
	"github.com/thetkpark/cscms-temp-storage/data/model"
	"gorm.io/gorm"
	"time"
)

type FileDataStore interface {
	Create(file *model.File) error
	FindByID(fileID string) (*model.File, error)
	FindByToken(token string) (*model.File, error)
	IncreaseVisited(id string) error
	FindByUserID(userId uint) (*[]model.File, error)
	DeleteByID(fileId string) error
	UpdateToken(fileID string, newToken string) error
}

type GormFileDataStore struct {
	db               *gorm.DB
	maxStoreDuration time.Duration
}

func NewGormFileDataStore(db *gorm.DB, duration time.Duration) (*GormFileDataStore, error) {
	if err := db.AutoMigrate(&model.File{}); err != nil {
		return nil, err
	}

	return &GormFileDataStore{
		db:               db,
		maxStoreDuration: duration,
	}, nil
}

func (store *GormFileDataStore) Create(file *model.File) error {
	tx := store.db.Create(file)
	return tx.Error
}

func (store *GormFileDataStore) FindByID(fileID string) (*model.File, error) {
	var file model.File
	tx := store.db.Where(&model.File{ID: fileID}).First(&file)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &file, tx.Error
}

func (store *GormFileDataStore) FindByToken(token string) (*model.File, error) {
	var files []*model.File
	if tx := store.db.Where(&model.File{Token: token}).Find(&files); tx.Error != nil {
		return nil, tx.Error
	}

	var file *model.File
	for _, v := range files {
		if v.ExpiredAt.UTC().After(time.Now().UTC()) {
			file = v
			break
		}
	}

	return file, nil
}

func (store *GormFileDataStore) FindByUserID(userId uint) (*[]model.File, error) {
	var files []model.File
	tx := store.db.Where(&model.File{UserID: userId}).Find(&files)
	return &files, tx.Error
}

func (store *GormFileDataStore) DeleteByID(fileId string) error {
	tx := store.db.Delete(&model.File{ID: fileId})
	return tx.Error
}

func (store *GormFileDataStore) IncreaseVisited(id string) error {
	tx := store.db.Table("files").Where(&model.File{ID: id}).UpdateColumns(map[string]interface{}{
		"visited":    gorm.Expr("visited + ?", 1),
		"updated_at": time.Now().UTC(),
	})
	return tx.Error
}

func (store *GormFileDataStore) UpdateToken(fileID string, newToken string) error {
	tx := store.db.Model(&model.File{}).Where("id", fileID).UpdateColumn("token", newToken)
	return tx.Error
}
