package model

// Stats is a struct with two fields, UrlsCount and UsersCount, both of which are integers.
type Stats struct {
	UrlsCount  int `json:"urls"`  // The number of URLs that have been shortened
	UsersCount int `json:"users"` // The number of registered users
}
