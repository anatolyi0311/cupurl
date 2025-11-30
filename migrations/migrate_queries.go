package migrations

const (
	createTable = `
		-- Создание таблицы
		CREATE TABLE IF NOT EXISTS cupurl
		(
			id BIGINT NOT NULL GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			originalURL varchar(255) NOT NULL UNIQUE,                                                                
			shortURL varchar(255)  NOT NULL UNIQUE                                            
		);

		-- Базовый индекс для поиска по originalURL
		CREATE INDEX idx_cupurl_originalURL ON cupurl(originalURL);
		-- Индекс для поиска по shortURL
		CREATE INDEX idx_cupurl_shortURL ON cupurl(shortURL);
	`

	dropTable = `
		-- Откат создания индексов на столбец shortURL / originalURL
		DROP INDEX IF EXISTS idx_cupurl_originalURL;
		DROP INDEX IF EXISTS idx_cupurl_shortURL;

		-- Откат добавления столбца originalURL / shortURL
		--ALTER TABLE cupurl DROP COLUMN IF EXISTS originalURL; 
		--ALTER TABLE cupurl DROP COLUMN IF EXISTS shortURL; 

		-- Откат создания таблицы
		--DROP TABLE IF EXISTS cupurl;
		DELETE FROM cupurl WHERE originalURL = "" 
		DELETE FROM cupurl WHERE shortURL = "" 
	`
)
