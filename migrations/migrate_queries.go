package migrations

/*
PostgreSQL:
Supports TRUE, FALSE, 't', 'f', 'y', 'n', 'yes', 'no', '1', '0' for boolean columns.
MySQL:
BOOL and BOOLEAN are aliases for TINYINT(1). It's recommended to use TRUE and FALSE literals or 1 and 0.
SQL Server:
Uses the BIT data type, which stores 0, 1, or NULL. You can assign 'true' or 'false' strings, which SQL Server implicitly converts to 1 or 0.
SQLite:
Stores boolean values as INTEGER, with 0 for false and 1 for true.
*/

const (
	createTable = `
		-- Создание таблицы
		CREATE TABLE IF NOT EXISTS cupurl
		(
			id BIGINT NOT NULL GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			originalURL varchar(255) NOT NULL UNIQUE,                                                                
			shortURL varchar(255) NOT NULL UNIQUE,                                         
			deletedFlag bool DEFAULT FALSE
		);

		-- Базовый индекс для поиска по originalURL
		--CREATE INDEX idx_cupurl_originalURL ON cupurl(originalURL);
		-- Индекс для поиска по shortURL
		--CREATE INDEX idx_cupurl_shortURL ON cupurl(shortURL);

		--ALTER TABLE cupurl ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN;
		--ALTER TABLE cupurl ADD COLUMN IF NOT EXISTS userID varchar(255);
		
		--CREATE INDEX idx_cupurl_isDeleted ON cupurl(is_deleted);
		--CREATE INDEX idx_cupurl_userID ON cupurl(userID);
	`

	dropTable = `
		-- Откат создания индексов на столбец shortURL / originalURL
		--DROP INDEX IF EXISTS idx_cupurl_originalURL;
		--DROP INDEX IF EXISTS idx_cupurl_shortURL;
		--DROP INDEX IF EXISTS idx_cupurl_isDeleted;
		--DROP INDEX IF EXISTS idx_cupurl_userID;

		-- Откат добавления столбца originalURL / shortURL
		--ALTER TABLE cupurl DROP COLUMN IF EXISTS originalURL; 
		--ALTER TABLE cupurl DROP COLUMN IF EXISTS shortURL; 

		-- Откат создания таблицы
		DROP TABLE IF EXISTS cupurl;

		--DELETE FROM cupurl WHERE originalURL = "" 
		--DELETE FROM cupurl WHERE shortURL = "" 
	`
)
