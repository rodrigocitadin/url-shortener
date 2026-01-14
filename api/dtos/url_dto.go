package dtos

type StoreUrlRequest struct {
	URL       string `json:"url"`
	Shortcode string `json:"shortcode"`
}

type StoreUrlResponse struct {
	ShortenedURL string `json:"shortened_url"`
}

type GetUrlRequest struct {
	Shortcode string `param:"shortcode"`
}
