package repository

const (
	querySetURL = `
		INSERT INTO cupurl (originalURL, shortURL, userID) 
        VALUES ($1, $2, $3)
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
	queryUpdateURL = `
		UPDATE cupurl 
		SET deletedFlag = true 
		WHERE shortURL = $1 AND userID = $2;
	`
	queryDeleteURL = `
		DELETE FROM cupurl 
		WHERE shortURL = $1 AND userID = $2 AND deletedFlag = true;
	`
)
