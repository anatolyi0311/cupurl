package model

type Err string

func (e Err) Error() string {
	return string(e)
}

const (
	ErrURLAlreadyExists Err = "url already exists"
	ErrEmptyUserID      Err = "empty user ID"
	ErrDeletedURL       Err = "url is deleted"
)
