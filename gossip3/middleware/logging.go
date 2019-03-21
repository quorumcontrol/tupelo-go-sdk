package middleware

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/AsynkronIT/protoactor-go/actor"
	"go.uber.org/zap"
)

var Log *zap.SugaredLogger
var globalLevel zap.AtomicLevel

func init() {
	globalLevel = zap.NewAtomicLevelAt(zap.WarnLevel)
	Log = buildLogger()
}

type LogAware interface {
	SetLog(log *zap.SugaredLogger)
}

type LogAwareHolder struct {
	Log *zap.SugaredLogger
}

func (state *LogAwareHolder) SetLog(log *zap.SugaredLogger) {
	state.Log = log
}

type LogPlugin struct{}

func (p *LogPlugin) OnStart(ctx actor.ReceiverContext) {
	if p, ok := ctx.Actor().(LogAware); ok {
		p.SetLog(Log.Named(ctx.Self().GetId()))
	}
}
func (p *LogPlugin) OnOtherMessage(ctx actor.ReceiverContext, env *actor.MessageEnvelope) {}

// Logger is message middleware which logs messages before continuing to the next middleware
func LoggingMiddleware(next actor.ReceiverFunc) actor.ReceiverFunc {
	fn := func(c actor.ReceiverContext, env *actor.MessageEnvelope) {
		// message := c.Message()
		// Log.Debugw("message", "id", c.Self(), "type", reflect.TypeOf(message), "sender", c.Sender().GetId()) //, "msg", message)
		next(c, env)
	}

	return fn
}

func SetLogLevel(level string) error {
	return globalLevel.UnmarshalText([]byte(level))
}

func buildLogger() *zap.SugaredLogger {
	cfg := zap.NewDevelopmentConfig()
	cfg.Level = globalLevel

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	sugar := logger.Sugar()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		// Ensure that log output gets written - it seems flushing to stderr produces an error message
		// related to ioctl, possibly not compatible with that target
		if err := sugar.Sync(); err != nil {
		}
		os.Exit(0)
	}()

	return sugar
}
