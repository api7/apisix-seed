package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.SugaredLogger

func init() {
	if env := os.Getenv("ENV"); env == "test" {
		InitLogger()
	}
}

func InitLogger() {
	logger = GetLogger(ErrorLog)
}

func GetLogger(logType Type) *zap.SugaredLogger {
	writeSyncer := fileWriter(logType)
	encoder := getEncoder(logType)
	logLevel := zapcore.InfoLevel
	if logType == ErrorLog {
		logLevel = zapcore.ErrorLevel
	}
	core := zapcore.NewCore(encoder, writeSyncer, logLevel)

	zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2))
	return zapLogger.Sugar()
}

func getEncoder(logType Type) zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	if logType == AccessLog {
		encoderConfig.LevelKey = zapcore.OmitKey
	}

	return zapcore.NewConsoleEncoder(encoderConfig)
}

func fileWriter(logType Type) zapcore.WriteSyncer {
	// standard output
	if logType == ErrorLog {
		return zapcore.Lock(os.Stderr)
	}
	return zapcore.Lock(os.Stdout)
}

func getZapFields(logger *zap.SugaredLogger, fields []interface{}) *zap.SugaredLogger {
	if len(fields) == 0 {
		return logger
	}
	return logger.With(fields)
}
