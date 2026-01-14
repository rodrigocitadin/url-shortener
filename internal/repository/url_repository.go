package repository

import (
	"github.com/rodrigocitadin/url-shortener/internal/entities"
	"gorm.io/gorm"
)

type URLRepository struct {
	db *gorm.DB
}

func NewURLRepository(db *gorm.DB) *URLRepository {
	return &URLRepository{db: db}
}

func (r *URLRepository) Save(urlEntity *entities.URLEntity) error {
	return r.db.Create(&urlEntity).Error
}

func (r *URLRepository) Find(shortCode string) (*entities.URLEntity, error) {
	var urlEntity entities.URLEntity
	err := r.db.Find(&urlEntity, "shortcode = ?", shortCode).Error
	return &urlEntity, err
}
