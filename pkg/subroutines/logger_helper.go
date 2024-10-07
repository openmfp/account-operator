package subroutines

import (
	"context"

	"github.com/openmfp/golang-commons/context/keys"
	"github.com/rs/zerolog"
)

func logInfo(ctx context.Context, msg string, err error) {
	if ctx == nil {
		return
	}
	loggerFromCtx := ctx.Value(keys.LoggerCtxKey)
	if loggerFromCtx != nil {
		return
	}
	logger, ok := loggerFromCtx.(zerolog.Logger)
	if !ok {
		return
	}
	logger.Info().Err(err).Msg(msg)
}
