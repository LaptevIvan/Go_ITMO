package log

import (
	"github.com/project/library/pkg/logger"
	"go.uber.org/zap"
)

func InfoRegisterAuthor(l *zap.Logger, msg string, traceID, authorName string, id ...string) {
	if len(id) == 0 {
		logger.MakeInfo(l, msg,
			zap.String("trace_id", traceID),
			zap.String("author_name", authorName),
			zap.String("action", RegisterAuthor))
		return
	}
	logger.MakeInfo(l, msg,
		zap.String("trace_id", traceID),
		zap.String("author_id", id[0]),
		zap.String("author_name", authorName),
		zap.String("action", RegisterAuthor))
}

func ErrorRegisterAuthor(l *zap.Logger, err error, msg string, traceID, authorName string) bool {
	return logger.CheckError(err, l, msg,
		zap.String("trace_id", traceID),
		zap.String("author_name", authorName),
		zap.Error(err),
		zap.String("action", RegisterAuthor))
}

func InfoChangeAuthorInfo(l *zap.Logger, msg string, traceID, authorID, newName string) {
	logger.MakeInfo(l, msg,
		zap.String("trace_id", traceID),
		zap.String("author_id", authorID),
		zap.String("author_name", newName),
		zap.String("action", ChangeAuthorInfo))
}

func ErrorChangeAuthorInfo(l *zap.Logger, err error, msg string, traceID, authorID, newName string) bool {
	return logger.CheckError(err, l, msg,
		zap.String("trace_id", traceID),
		zap.String("author_id", authorID),
		zap.String("author_name", newName),
		zap.Error(err),
		zap.String("action", ChangeAuthorInfo))
}

func InfoGetAuthorInfo(l *zap.Logger, msg string, traceID, authorID string) {
	logger.MakeInfo(l, msg,
		zap.String("trace_id", traceID),
		zap.String("author_id", authorID),
		zap.String("action", GetAuthorInfo))
}

func ErrorGetAuthorInfo(l *zap.Logger, err error, msg string, traceID, authorID string) bool {
	return logger.CheckError(err, l, msg,
		zap.String("trace_id", traceID),
		zap.String("author_id", authorID),
		zap.Error(err),
		zap.String("action", GetAuthorInfo))
}
