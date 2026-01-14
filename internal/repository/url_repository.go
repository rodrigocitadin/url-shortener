package repository

import "gorm.io/gorm"

type URLRepository struct {
	db *gorm.DB
}

func NewURLRepository(db *gorm.DB) *URLRepository {
	return &URLRepository{db: db}
}

func (r *URLRepository) Save(shortCode, originalURL string) error {
	query := `INSERT INTO urls (shortcode, url) VALUES (?, ?)`
	err := r.db.Raw(query, shortCode, originalURL).Error

	return err
}

func (r *URLRepository) Find(shortCode string) (string, error) {
	var originalURL string
	query := `SELECT url FROM urls WHERE shortcode = ?`

	err := r.db.Raw(query, shortCode).Scan(&originalURL).Error
	return originalURL, err
}
