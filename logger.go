package main

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var slogger *zap.SugaredLogger

func InitLogger() {
	writeSyncer := getLogWriter()
	encoder := getEncoder()
	core := zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel)
	// Print function lines
	logger := zap.New(core, zap.AddCaller())
	slogger = logger.Sugar()
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	// The format time can be customized
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// Save file log cut
func getLogWriter() zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   "udp-forwarder.log", // Log name
		MaxSize:    1,                   // File content size, MB
		MaxBackups: 5,                   // Maximum number of old files retained
		MaxAge:     30,                  // Maximum number of days to keep old files
		Compress:   false,               // Is the file compressed
	}
	return zapcore.AddSync(lumberJackLogger)
}
