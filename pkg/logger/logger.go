/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package logger

import (
	"context"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/rs/zerolog"
)

type LoggingService interface {
	Err(msg string, metadata map[string]any, err error)
	ErrCtx(ctx context.Context, msg string, metadata map[string]any, err error)
	Info(msg string, metadata map[string]any)
	InfoCtx(ctx context.Context, msg string, metadata map[string]any)
	Panic(msg string, metadata map[string]any)
	PanicCtx(ctx context.Context, msg string, metadata map[string]any)
	Debug(msg string, ctx map[string]any)
	DebugCtx(ctx context.Context, msg string, metadata map[string]any)
}

var globalLogger *zeroLogger

type zeroLogger struct {
	logger  *zerolog.Logger
	service string
}

func GetLogger() *zerolog.Logger {
	if globalLogger == nil {
		InitLogger("info", "", os.Stdout)
	}
	return globalLogger.logger
}

func InitLogger(level, service string, logOutput io.Writer) {
	levelMap := map[string]zerolog.Level{
		"info":  zerolog.InfoLevel,
		"debug": zerolog.DebugLevel,
		"warn":  zerolog.WarnLevel,
		"error": zerolog.WarnLevel,
	}
	zlevel, ok := levelMap[level]
	if !ok {
		zlevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(zlevel)

	logger := zerolog.New(logOutput).With().Timestamp().Caller().Logger()
	commit := func() string {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					return setting.Value
				}
			}
		}

		return "no commit available"
	}()
	logger = logger.Hook(ContextHook{commit: commit}).Hook(FunctionNameHook{})

	globalLogger = &zeroLogger{&logger, service}
}

func (l *zeroLogger) Err(msg string, metadata map[string]any, err error) {
	l.logger.Err(err).Fields(metadata).Msg(msg)
}

func (l *zeroLogger) ErrCtx(ctx context.Context, msg string, metadata map[string]any, err error) {
	l.logger.Err(err).Ctx(ctx).Fields(metadata).Msg(msg)
}

func (l *zeroLogger) Info(msg string, metadata map[string]any) {
	l.logger.Info().Fields(metadata).Msg(msg)
}

func (l *zeroLogger) InfoCtx(ctx context.Context, msg string, metadata map[string]any) {
	l.logger.Info().Ctx(ctx).Fields(metadata).Msg(msg)
}

func (l *zeroLogger) Panic(msg string, metadata map[string]any) {
	l.logger.Panic().Fields(metadata).Msg(msg)
}

func (l *zeroLogger) PanicCtx(ctx context.Context, msg string, metadata map[string]any) {
	l.logger.Panic().Ctx(ctx).Fields(metadata).Msg(msg)
}

func (l *zeroLogger) Debug(msg string, metadata map[string]any) {
	l.logger.Debug().Fields(metadata).Msg(msg)
}

func (l *zeroLogger) DebugCtx(ctx context.Context, msg string, metadata map[string]any) {
	l.logger.Debug().Ctx(ctx).Fields(metadata).Msg(msg)
}

type ContextHook struct {
	commit string
}

func (h ContextHook) Run(e *zerolog.Event, _ zerolog.Level, _ string) {
	if h.commit != "" {
		e.Str("git.commit", h.commit)
	}
	ctx := e.GetCtx()
	for _, key := range DebugContextKeys {
		v := ctx.Value(key)
		if v != nil {
			e.Str(key, v.(string))
		}
	}
}

// FunctionNameHook is a zerolog.Hook that adds the function name to the logger's context.
type FunctionNameHook struct{}

// Run is called for every log entry.
func (h FunctionNameHook) Run(e *zerolog.Event, _ zerolog.Level, _ string) {
	// Get the function name using the runtime package
	methodName := "Unknown"
	pc, _, _, _ := runtime.Caller(3) //nolint:dogsled
	callerFunc := runtime.FuncForPC(pc)
	if callerFunc != nil {
		// Retrieve the name of the caller function
		splitName := strings.Split(callerFunc.Name(), ".")
		methodName = splitName[len(splitName)-1]
	}

	// Add function name to the logger's context
	e.Str("function", methodName)
}

// Levels returns the log levels that this hook should be enabled for.
func (h FunctionNameHook) Levels() []zerolog.Level {
	return []zerolog.Level{
		zerolog.InfoLevel,
		zerolog.ErrorLevel,
		zerolog.WarnLevel,
		zerolog.FatalLevel,
		zerolog.DebugLevel,
	}
}
