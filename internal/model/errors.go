package model

type Err string

func (e Err) Error() string {
	return string(e)
}

const ErrURLAlreadyExists Err = "url already exists"
