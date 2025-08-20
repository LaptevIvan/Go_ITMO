package log

import (
	"github.com/project/library/pkg/logger"
	"go.uber.org/zap"
)

func InfoAddBook(l *zap.Logger, msg string, traceID, name string, authorIDs []string, id ...string) {
	if len(id) == 0 {
		logger.MakeInfo(l, msg,
			zap.String("trace_id", traceID),
			zap.String("book_name", name),
			zap.Strings("book_authors", authorIDs),
			zap.String("action", AddBook))
		return
	}
	logger.MakeInfo(l, "book was added",
		zap.String("trace_id", traceID),
		zap.String("book_id", id[0]),
		zap.String("book_name", name),
		zap.Strings("book_authors", authorIDs),
		zap.String("action", AddBook))
}

func ErrorAddBook(l *zap.Logger, err error, msg string, traceID, name string, authorIDs []string) bool {
	return logger.CheckError(err, l, msg,
		zap.String("trace_id", traceID),
		zap.String("book_name", name),
		zap.Strings("book_authors", authorIDs),
		zap.Error(err),
		zap.String("action", AddBook))
}

func InfoGetBookInfo(l *zap.Logger, msg string, traceID, bookID string) {
	logger.MakeInfo(l, msg,
		zap.String("trace_id", traceID),
		zap.String("book_id", bookID),
		zap.String("action", GetBookInfo))
}

func ErrorGetBookInfo(l *zap.Logger, err error, msg string, traceID, bookID string) bool {
	return logger.CheckError(err, l, msg,
		zap.String("trace_id", traceID),
		zap.String("book_id", bookID),
		zap.Error(err),
		zap.String("action", GetBookInfo))
}

func InfoUpdateBook(l *zap.Logger, msg string, traceID, id, newName string, newAuthorIDs []string) {
	logger.MakeInfo(l, msg,
		zap.String("trace_id", traceID),
		zap.String("book_id", id),
		zap.String("book_name", newName),
		zap.Strings("book_authors", newAuthorIDs),
		zap.String("action", UpdateBook))
}

func ErrorUpdateBook(l *zap.Logger, err error, msg string, traceID, id, newName string, newAuthorIDs []string) bool {
	return logger.CheckError(err, l, msg,
		zap.String("trace_id", traceID),
		zap.String("book_id", id),
		zap.String("book_name", newName),
		zap.Strings("book_authors", newAuthorIDs),
		zap.Error(err),
		zap.String("action", UpdateBook))
}

func InfoGetAuthorBooks(l *zap.Logger, msg string, traceID, authorID string) {
	logger.MakeInfo(l, msg,
		zap.String("trace_id", traceID),
		zap.String("author_id", authorID),
		zap.String("action", GetAuthorBooks))
}

func ErrorGetAuthorBooks(l *zap.Logger, err error, msg string, traceID, authorID string) bool {
	return logger.CheckError(err, l, msg,
		zap.String("trace_id", traceID),
		zap.String("author_id", authorID),
		zap.Error(err),
		zap.String("action", GetAuthorBooks))
}

func InfoSendBook(l *zap.Logger, msg string, traceID, bookID, authorID string) {
	logger.MakeInfo(l, msg,
		zap.String("trace_id", traceID),
		zap.String("book_id", bookID),
		zap.String("author_id", authorID),
		zap.String("action", GetAuthorBooks))
}

func ErrorSendBook(l *zap.Logger, err error, msg string, traceID, bookID, authorID string) bool {
	return logger.CheckError(err, l, msg,
		zap.String("trace_id", traceID),
		zap.String("book_id", bookID),
		zap.String("author_id", authorID),
		zap.Error(err),
		zap.String("action", GetAuthorBooks))
}
