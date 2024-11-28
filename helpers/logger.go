package helpers

import (
	"context"

	"github.com/rs/zerolog/log"
)

func LogInfo(msg string, extra map[string]any, ctx context.Context) {
	logEvent := log.Info()
	for key, value := range extra {
		logEvent.Interface(key, value)
	}
	if ctx == nil {
		logEvent.Msg(msg)
		return
	}
	if apiName, ok := ctx.Value(APINameKey{}).(string); ok {
		logEvent.Interface("apiName", apiName)
	}
	logEvent.Msg(msg)
}

func LogError(err error, msg string, extra map[string]any, ctx context.Context) {
	logEvent := log.Error().Err(err)
	for key, value := range extra {
		logEvent.Interface(key, value)
	}
	if ctx == nil {
		logEvent.Msg(msg)
		return
	}
	if apiName, ok := ctx.Value(APINameKey{}).(string); ok {
		logEvent.Interface("apiName", apiName)
	}
	logEvent.Msg(msg)
}
