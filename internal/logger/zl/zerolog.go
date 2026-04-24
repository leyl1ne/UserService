package zl

import (
	"io"
	"os"

	"github.com/leyl1ne/UserService/internal/logger"
	"github.com/rs/zerolog"
)

type ZerologLogger struct {
	log zerolog.Logger
}

func NewZerologLogger(level string, out io.Writer) *ZerologLogger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)

	if out == nil {
		out = zerolog.ConsoleWriter{Out: os.Stderr}
	}

	l := zerolog.New(out).With().Timestamp().Logger()
	return &ZerologLogger{log: l}

}

func (z *ZerologLogger) With(fields ...logger.Field) logger.Logger {
	if len(fields) == 0 {
		return z
	}

	ctx := z.log.With()

	for _, f := range fields {
		ctx = ctx.Interface(f.Key, f.Value)
	}

	newLogger := ctx.Logger()

	return &ZerologLogger{
		log: newLogger,
	}
}

func (z *ZerologLogger) Info(msg string, fields ...logger.Field) {
	z.log.Info().Fields(toZerologFields(fields)).Msg(msg)
}

func (z *ZerologLogger) Warn(msg string, fields ...logger.Field) {
	z.log.Warn().Fields(toZerologFields(fields)).Msg(msg)
}

func (z *ZerologLogger) Debug(msg string, fields ...logger.Field) {
	z.log.Debug().Fields(toZerologFields(fields)).Msg(msg)
}

func (z *ZerologLogger) Error(msg string, fields ...logger.Field) {
	z.log.Error().Fields(toZerologFields(fields)).Msg(msg)
}

func (z *ZerologLogger) Fatal(msg string, fields ...logger.Field) {
	z.log.Fatal().Fields(toZerologFields(fields)).Msg(msg)
}

func (z *ZerologLogger) Panic(msg string, fields ...logger.Field) {
	z.log.Panic().Fields(toZerologFields(fields)).Msg(msg)
}

func toZerologFields(fields []logger.Field) map[string]interface{} {
	if len(fields) == 0 {
		return nil
	}

	m := make(map[string]interface{}, len(fields))
	for _, f := range fields {
		m[f.Key] = f.Value
	}
	return m
}
