package logger

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog"
	"go.uber.org/fx/fxevent"
)

type FxLogger struct {
	log zerolog.Logger
}

func NewFxLogger(log zerolog.Logger) FxLogger {
	return FxLogger{log: log.With().Str("component", "fx").Logger()}
}

func (l FxLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuting:
		l.log.Debug().
			Str("callee", e.FunctionName).
			Str("caller", e.CallerName).
			Msg("fx on_start executing")
	case *fxevent.OnStartExecuted:
		if e.Err != nil {
			l.log.Error().
				Err(e.Err).
				Str("callee", e.FunctionName).
				Str("caller", e.CallerName).
				Dur("runtime", e.Runtime).
				Msg("fx on_start failed")
			return
		}
		l.log.Debug().
			Str("callee", e.FunctionName).
			Str("caller", e.CallerName).
			Dur("runtime", e.Runtime).
			Msg("fx on_start executed")
	case *fxevent.OnStopExecuting:
		l.log.Debug().
			Str("callee", e.FunctionName).
			Str("caller", e.CallerName).
			Msg("fx on_stop executing")
	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			l.log.Error().
				Err(e.Err).
				Str("callee", e.FunctionName).
				Str("caller", e.CallerName).
				Dur("runtime", e.Runtime).
				Msg("fx on_stop failed")
			return
		}
		l.log.Debug().
			Str("callee", e.FunctionName).
			Str("caller", e.CallerName).
			Dur("runtime", e.Runtime).
			Msg("fx on_stop executed")
	case *fxevent.Supplied:
		if e.Err != nil {
			entry := l.log.Error().
				Err(e.Err).
				Str("type", e.TypeName).
				Str("stacktrace", strings.Join(e.StackTrace, " | "))
			if e.ModuleName != "" {
				entry = entry.Str("module", e.ModuleName)
			}
			entry.Msg("fx supply failed")
			return
		}
		entry := l.log.Debug().
			Str("type", e.TypeName).
			Str("stacktrace", strings.Join(e.StackTrace, " | "))
		if e.ModuleName != "" {
			entry = entry.Str("module", e.ModuleName)
		}
		entry.Msg("fx supplied")
	case *fxevent.Provided:
		if e.Err != nil {
			entry := l.log.Error().
				Err(e.Err).
				Strs("outputs", e.OutputTypeNames).
				Str("constructor", e.ConstructorName).
				Str("stacktrace", strings.Join(e.StackTrace, " | "))
			if e.ModuleName != "" {
				entry = entry.Str("module", e.ModuleName)
			}
			if e.Private {
				entry = entry.Bool("private", true)
			}
			entry.Msg("fx provide failed")
			return
		}
		entry := l.log.Debug().
			Strs("outputs", e.OutputTypeNames).
			Str("constructor", e.ConstructorName).
			Str("stacktrace", strings.Join(e.StackTrace, " | "))
		if e.ModuleName != "" {
			entry = entry.Str("module", e.ModuleName)
		}
		if e.Private {
			entry = entry.Bool("private", true)
		}
		entry.Msg("fx provided")
	case *fxevent.Replaced:
		if e.Err != nil {
			entry := l.log.Error().
				Err(e.Err).
				Strs("outputs", e.OutputTypeNames).
				Str("stacktrace", strings.Join(e.StackTrace, " | "))
			if e.ModuleName != "" {
				entry = entry.Str("module", e.ModuleName)
			}
			entry.Msg("fx replace failed")
			return
		}
		entry := l.log.Debug().
			Strs("outputs", e.OutputTypeNames).
			Str("stacktrace", strings.Join(e.StackTrace, " | "))
		if e.ModuleName != "" {
			entry = entry.Str("module", e.ModuleName)
		}
		entry.Msg("fx replaced")
	case *fxevent.Decorated:
		if e.Err != nil {
			entry := l.log.Error().
				Err(e.Err).
				Strs("outputs", e.OutputTypeNames).
				Str("decorator", e.DecoratorName).
				Str("stacktrace", strings.Join(e.StackTrace, " | "))
			if e.ModuleName != "" {
				entry = entry.Str("module", e.ModuleName)
			}
			entry.Msg("fx decorate failed")
			return
		}
		entry := l.log.Debug().
			Strs("outputs", e.OutputTypeNames).
			Str("decorator", e.DecoratorName).
			Str("stacktrace", strings.Join(e.StackTrace, " | "))
		if e.ModuleName != "" {
			entry = entry.Str("module", e.ModuleName)
		}
		entry.Msg("fx decorated")
	case *fxevent.Run:
		if e.Err != nil {
			l.log.Error().
				Err(e.Err).
				Str("name", e.Name).
				Str("kind", e.Kind).
				Str("module", e.ModuleName).
				Msg("fx run failed")
			return
		}
		l.log.Debug().
			Str("name", e.Name).
			Str("kind", e.Kind).
			Str("module", e.ModuleName).
			Msg("fx run")
	case *fxevent.Invoking:
		entry := l.log.Debug().Str("function", e.FunctionName)
		if e.ModuleName != "" {
			entry = entry.Str("module", e.ModuleName)
		}
		entry.Msg("fx invoking")
	case *fxevent.Invoked:
		if e.Err != nil {
			entry := l.log.Error().
				Err(e.Err).
				Str("function", e.FunctionName)
			if e.ModuleName != "" {
				entry = entry.Str("module", e.ModuleName)
			}
			entry.Msg("fx invoke failed")
			return
		}
		entry := l.log.Debug().Str("function", e.FunctionName)
		if e.ModuleName != "" {
			entry = entry.Str("module", e.ModuleName)
		}
		entry.Msg("fx invoked")
	case *fxevent.Stopping:
		l.log.Info().Str("signal", strings.ToUpper(e.Signal.String())).Msg("fx stopping")
	case *fxevent.Stopped:
		if e.Err != nil {
			l.log.Error().Err(e.Err).Msg("fx stopped with error")
			return
		}
		l.log.Info().Msg("fx stopped")
	case *fxevent.RollingBack:
		l.log.Warn().Err(e.StartErr).Msg("fx rolling back")
	case *fxevent.RolledBack:
		if e.Err != nil {
			l.log.Error().Err(e.Err).Msg("fx rollback failed")
			return
		}
		l.log.Warn().Msg("fx rolled back")
	case *fxevent.Started:
		if e.Err != nil {
			l.log.Error().Err(e.Err).Msg("fx start failed")
			return
		}
		l.log.Info().Msg("fx started")
	case *fxevent.LoggerInitialized:
		if e.Err != nil {
			l.log.Error().
				Err(e.Err).
				Str("constructor", e.ConstructorName).
				Msg("fx logger initialization failed")
			return
		}
		l.log.Debug().
			Str("constructor", e.ConstructorName).
			Msg("fx logger initialized")
	default:
		l.log.Debug().Str("event", fmt.Sprintf("%T", event)).Msg("fx event")
	}
}
