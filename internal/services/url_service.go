package services

import (
	"github.com/rodrigocitadin/url-shortener/internal/entities"
	"github.com/rodrigocitadin/url-shortener/internal/repository"
)

type URLService interface {
	Get(shortcode string) (*entities.URLEntity, error)
	Store(url, shortcode string) error
}

type urlService struct {
	uow repository.UnitOfWork
}

func (u *urlService) Get(shortcode string) (*entities.URLEntity, error) {
	r := u.uow.URLS(shortcode)
	return r.Find(shortcode)
}

func (u *urlService) Store(url string, shortcode string) error {
	r := u.uow.URLS(shortcode)
	return r.Save(&entities.URLEntity{
		URL:       url,
		Shortcode: shortcode,
	})
}

func NewURLService(uow repository.UnitOfWork) URLService {
	return &urlService{uow: uow}
}
