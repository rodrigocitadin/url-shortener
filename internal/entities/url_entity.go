package entities

type URLEntity struct {
	ID        int64
	Shortcode string
	URL       string
	Accesses  int64
}

func (URLEntity) TableName() string {
	return "urls"
}
