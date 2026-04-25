package bootstrap

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"traffic-monitor/internal/config"
	embeddedweb "traffic-monitor/web"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

func NewGinEngine(appConfig config.AppConfig, log zerolog.Logger) (*gin.Engine, error) {
	if appConfig.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(requestLogger(log))

	staticFS, err := embeddedweb.DistFS()
	if err != nil {
		return nil, err
	}

	registerStaticRoutes(engine, staticFS)

	return engine, nil
}

func NewHTTPServer(engine *gin.Engine, cfg config.HTTPConfig) *http.Server {
	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
	}
}

func RegisterHTTPServer(lifecycle fx.Lifecycle, server *http.Server, cfg config.HTTPConfig, log zerolog.Logger) {
	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				log.Info().Str("addr", cfg.Addr).Msg("http server started")
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Error().Err(err).Msg("http server stopped unexpectedly")
				}
			}()

			return nil
		},
		OnStop: func(stopContext context.Context) error {
			shutdownContext, cancel := context.WithTimeout(stopContext, cfg.StopTimeout)
			defer cancel()

			if err := server.Shutdown(shutdownContext); err != nil {
				return fmt.Errorf("shutdown http server: %w", err)
			}

			log.Info().Msg("http server stopped")
			return nil
		},
	})
}

func registerStaticRoutes(engine *gin.Engine, staticFS fs.FS) {
	assetsFS, err := fs.Sub(staticFS, "assets")
	if err != nil {
		engine.NoRoute(func(ctx *gin.Context) {
			ctx.String(http.StatusInternalServerError, "static assets unavailable")
		})
		return
	}

	engine.StaticFS("/assets", http.FS(assetsFS))
	engine.NoRoute(func(ctx *gin.Context) {
		indexContent, err := fs.ReadFile(staticFS, "index.html")
		if err != nil {
			ctx.String(http.StatusInternalServerError, "static index unavailable")
			return
		}

		ctx.Data(http.StatusOK, "text/html; charset=utf-8", indexContent)
	})
}

func requestLogger(log zerolog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startedAt := time.Now()
		ctx.Next()

		log.Info().
			Int("status", ctx.Writer.Status()).
			Str("method", ctx.Request.Method).
			Str("path", ctx.Request.URL.Path).
			Dur("duration", time.Since(startedAt)).
			Msg("http request completed")
	}
}
