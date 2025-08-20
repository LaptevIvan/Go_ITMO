package logger

import "go.uber.org/zap"

func CheckError(err error, logger *zap.Logger, msg string, fields ...zap.Field) bool {
	if err != nil {
		if logger != nil {
			logger.Error(msg, fields...)
		}
		return true
	}
	return false
}

func MakeInfo(logger *zap.Logger, msg string, fields ...zap.Field) {
	if logger != nil {
		logger.Info(msg, fields...)
	}
}
