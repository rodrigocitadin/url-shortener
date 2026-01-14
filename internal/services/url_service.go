package services

import "github.com/rodrigocitadin/url-shortener/internal/entities"

type urlService struct {
}

type URLService interface {
	Get(shortcode string) (entities.URLEntity, error)
	Store(url, shortcode string) error
}
