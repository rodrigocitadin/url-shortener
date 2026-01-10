package repository

type URLRepository struct {
	sharder *ShardManager
}

func (r *URLRepository) Save(shortCode, originalURL string) error {
	db := r.sharder.GetShard(shortCode)

	query := `INSERT INTO urls (short_code, original_url) VALUES ($1, $2)`
	_, err := db.Exec(query, shortCode, originalURL)

	return err
}

func (r *URLRepository) Find(shortCode string) (string, error) {
	db := r.sharder.GetShard(shortCode)

	var originalURL string
	query := `SELECT original_url FROM urls WHERE short_code = $1`

	err := db.QueryRow(query, shortCode).Scan(&originalURL)
	return originalURL, err
}
