package repository

const (
	querySetURL = `
		INSERT INTO cupurl (originalURL, shortURL) 
        VALUES ($1, $2)
        ON CONFLICT (originalURL) 
        DO NOTHING;
	`
	queryGetURL = `
		SELECT originalURL, deletedFlag 
		FROM cupurl
		WHERE shortURL = $1;
	`
	queryGetArrayURL = `
		SELECT originalURL, shortURL 
		FROM cupurl;
	`
	queryDeleteURL = `
		UPDATE cupurl 
		SET deletedFlag = true 
		WHERE shortURL = $1;
	`
)
