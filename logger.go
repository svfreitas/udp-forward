package main

import (
	"net/http"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var slogger *zap.SugaredLogger

func InitLogger(logFileLocation string, logFileSize int, logFileMaxBackups int) http.Handler {
	writeSyncer := getLogWriter(logFileLocation, logFileSize, logFileMaxBackups)
	encoder := getEncoder()
	atom := zap.NewAtomicLevel()
	core := zapcore.NewCore(encoder, writeSyncer, atom)
	// Print function lines

	logger := zap.New(core, zap.AddCaller())
	slogger = logger.Sugar()
	return atom
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	// The format time can be customized
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// Save file log cut
func getLogWriter(logfileLocation string, logFileSize int, logFileMaxBackups int) zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   logfileLocation + "udp-forwarder.log", // Log name
		MaxSize:    logFileSize,                           // File content size, MB
		MaxBackups: logFileMaxBackups,                     // Maximum number of old files retained
		//MaxAge:     90,                                    // Maximum number of days to keep old files
		Compress: false, // Is the file compressed
	}
	return zapcore.AddSync(lumberJackLogger)
}
