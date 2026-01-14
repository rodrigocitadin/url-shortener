package dtos

type StoreUrlRequest struct {
	URL       string `json:"url"`
	Shortcode string `json:"shortcode"`
}

type GetUrlRequest struct {
	Shortcode string `param:"shortcode"`
}
