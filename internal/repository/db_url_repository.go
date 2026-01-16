package repository

import (
	"github.com/rodrigocitadin/url-shortener/internal/entities"
	"gorm.io/gorm"
)

type URLRepository interface {
	Save(urlEntity *entities.URLEntity) error
	Find(shortCode string) (*entities.URLEntity, error)
}

type urlRepository struct {
	db *gorm.DB
}

func NewDatabaseURLRepository(db *gorm.DB) URLRepository {
	return &urlRepository{db: db}
}

func (c *urlRepository) Save(urlEntity *entities.URLEntity) error {
	return c.db.Create(&urlEntity).Error
}

func (r *urlRepository) Find(shortCode string) (*entities.URLEntity, error) {
	var urlEntity entities.URLEntity
	err := r.db.Find(&urlEntity, "shortcode = ?", shortCode).Error
	return &urlEntity, err
}
