package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.SugaredLogger

func init() {
	logger = GetLogger()
}

func GetLogger() *zap.SugaredLogger {
	// standard output
	writeSyncer := zapcore.Lock(os.Stderr)
	encoder := getEncoder()
	logLevel := zapcore.ErrorLevel
	core := zapcore.NewCore(encoder, writeSyncer, logLevel)

	zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2))
	return zapLogger.Sugar()
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getZapFields(logger *zap.SugaredLogger, fields []interface{}) *zap.SugaredLogger {
	if len(fields) == 0 {
		return logger
	}
	return logger.With(fields)
}
