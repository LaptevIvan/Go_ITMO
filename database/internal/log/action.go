package log

type Action = string

const (
	RegisterAuthor   Action = "RegisterAuthor"
	ChangeAuthorInfo        = "ChangeAuthorInfo"
	GetAuthorInfo           = "GetAuthorInfo"
	AddBook                 = "AddBook"
	GetBookInfo             = "GetBookInfo"
	UpdateBook              = "UpdateBook"
	GetAuthorBooks          = "GetAuthorBooks"
)
