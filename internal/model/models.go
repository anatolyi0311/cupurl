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
