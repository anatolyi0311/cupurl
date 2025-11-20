package repository

const (
	querySetURL = `
		INSERT INTO cupurl (originalURL, shortURL)
		VALUES ($1, $2)
		ON CONFLICT (originalURL) DO NOTHING;
	`
	queryGetURL = `
		SELECT originalURL
		FROM cupurl
		WHERE shortURL = $1
	`
)
