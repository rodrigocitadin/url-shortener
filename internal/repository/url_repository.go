package repository

type URLRepository struct {
	sharder *ShardManager
}

func NewURLRepository(sharder *ShardManager) *URLRepository {
	return &URLRepository{sharder: sharder}
}

func (r *URLRepository) Save(shortCode, originalURL string) error {
	db := r.sharder.GetShard(shortCode)

	query := `INSERT INTO urls (shortcode, url) VALUES ($1, $2)`
	_, err := db.Exec(query, shortCode, originalURL)

	return err
}

func (r *URLRepository) Find(shortCode string) (string, error) {
	db := r.sharder.GetShard(shortCode)

	var originalURL string
	query := `SELECT url FROM urls WHERE shortcode = $1`

	err := db.QueryRow(query, shortCode).Scan(&originalURL)
	return originalURL, err
}
