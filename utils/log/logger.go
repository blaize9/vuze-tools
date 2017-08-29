package log

import (
	"fmt"
	"github.com/blaize9/vuze-tools/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var logger *zap.Logger
var sugar *zap.SugaredLogger

func Init(environment string) {
	switch environment {
	case "DEVELOPMENT":
		InitLogToStdoutDebug()
	case "DEV":
		InitLogToStdoutDebug()

	case "PRODUCTION-STDOUT":
		InitLogToStdout()
	case "PROD-STDOUT":
		InitLogToStdout()

	case "PRODUCTION-JSON":
		InitLogToJsonFile()
		fmt.Printf("Outputting Log to %s\n", config.Get().Log.ErrorLogFilePath+config.Get().Log.ErrorLogFileExtension)
	case "PROD-JSON":
		InitLogToJsonFile()
		fmt.Printf("Outputting Log to %s\n", config.Get().Log.ErrorLogFilePath+config.Get().Log.ErrorLogFileExtension)

	case "PRODUCTION":
		InitLogToFile()
		fmt.Printf("Outputting Log to %s.log\n", config.Get().Log.ErrorLogFilePath)
	case "PROD":
		InitLogToFile()
		fmt.Printf("Outputting Log to %s.log\n", config.Get().Log.ErrorLogFilePath)

	default:
		InitLogToStdoutDebug()
	}
	Debugf("Environment: %s\n", environment)
}

func InitLogToStdoutDebug() {
	configZ := zap.NewDevelopmentConfig()
	configZ.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, _ = configZ.Build()
	sugar = logger.Sugar()
}

func InitLogToStdout() {
	configZ := zap.NewDevelopmentConfig()
	configZ.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	configZ.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, _ = configZ.Build()
	sugar = logger.Sugar()
}

func InitLogToFile() {
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   config.Get().Log.ErrorLogFilePath + ".log",
		MaxSize:    config.Get().Log.ErrorLogMaxSize, // megabytes
		MaxBackups: config.Get().Log.ErrorLogMaxBackups,
		MaxAge:     config.Get().Log.ErrorLogMaxAge, // days
	})
	configZ := zap.NewProductionEncoderConfig()
	configZ.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(configZ),
		w,
		zap.InfoLevel,
	)
	logger = zap.New(core)
	sugar = logger.Sugar()
}

func InitLogToJsonFile() {
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   config.Get().Log.ErrorLogFilePath + config.Get().Log.ErrorLogFileExtension,
		MaxSize:    config.Get().Log.ErrorLogMaxSize, // megabytes
		MaxBackups: config.Get().Log.ErrorLogMaxBackups,
		MaxAge:     config.Get().Log.ErrorLogMaxAge, // days
	})
	configZ := zap.NewProductionEncoderConfig()
	configZ.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(configZ),
		w,
		zap.InfoLevel,
	)
	logger = zap.New(core)
	sugar = logger.Sugar()
}

func Debug(msg string) {
	logger.Debug(msg)
}

func Debugf(msg string, args ...interface{}) {
	sugar.Debugf(msg, args...)
}

func Info(msg string) {
	logger.Info(msg)
}

func Infof(msg string, args ...interface{}) {
	sugar.Infof(msg, args...)
}

func Warn(msg string) {
	logger.Warn(msg)
}

func Warnf(msg string, args ...interface{}) {
	sugar.Warnf(msg, args...)
}

func Error(msg string) {
	logger.Error(msg)
}

func Errorf(msg string, args ...interface{}) {
	sugar.Errorf(msg, args...)
}

func Fatal(msg string) {
	logger.Fatal(msg)
}

func Fatalf(msg string, args ...interface{}) {
	sugar.Fatalf(msg, args...)
}

func Panic(msg string) {
	logger.Panic(msg)
}

func Panicf(msg string, args ...interface{}) {
	sugar.Panicf(msg, args...)
}
