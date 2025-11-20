package migrations

const (
	createTable = `
		CREATE TABLE IF NOT EXISTS cupurl
		(
			id BIGINT NOT NULL GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			originalURL varchar(255) NOT NULL UNIQUE,                                                                
			shortURL varchar(255)  NOT NULL UNIQUE                                            
		);
	`

	dropTable = `
		DROP TABLE IF EXISTS cupurl;
	`
)
