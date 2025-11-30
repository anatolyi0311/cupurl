package model

type SetURLJsonRequest struct {
	URL string `json:"url"`
}

type SetURLJsonResponse struct {
	URL string `json:"result"`
}

type SetArrayURLRequest struct {
	ID          string `json:"correlation_id"`
	OriginalURL string `json:"original_url"`
	ShortURL    string
}

type SetArrayURLResponse struct {
	ID  string `json:"correlation_id"`
	URL string `json:"short_url"`
}

type GetArrayURLRequest struct {
	// ID          string `json:"correlation_id"`
	OriginalURL string `json:"original_url"`
	ShortURL    string `json:"short_url"`
	UserID      string `json:"user_id"`
	DeletedFlag bool   `json:"is_deleted"` // is used to mark a record as deleted
	Hash        string
}

type GetArrayURLResponse struct {
	Short    string `json:"short_url"`
	Original string `json:"original_url"`
}

type ShortURL struct {
	ID          string `json:"correlation_id"` // Hash
	OriginalURL string `json:"original_url"`
	ShortURL    string `json:"short_url"`
	UserID      string `json:"user_id"`
	CreatedByID string `json:"created_by"` // ID of the user who created the short URL
	DeletedFlag bool   `json:"is_deleted"`
}

// ShortURL is main entity for system.
// type ShortURL struct {
// 	DeletedAt     time.Time `json:"deleted_at"`     // is used to mark a record as deleted
// 	OriginalURL   string    `json:"url"`            // original URL that was shortened
// 	ID            string    `json:"id"`             // unique ID of the short URL.
// 	CreatedByID   string    `json:"created_by"`     // ID of the user who created the short URL
// 	CorrelationID string    `json:"correlation_id"` // CorrelationID is used for matching original and shorten urls in shorten batch operation
// }
