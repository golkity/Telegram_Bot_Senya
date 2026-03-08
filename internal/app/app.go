package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"telegram_bot/internal/bot"
	"telegram_bot/internal/bot/handlers"
	"telegram_bot/internal/infra/excel"
	"telegram_bot/internal/infra/redis"
	"telegram_bot/internal/infra/s3"
	"telegram_bot/internal/infra/telegram"
	"telegram_bot/internal/infra/word"
	"telegram_bot/internal/modules/cms"
	"telegram_bot/internal/modules/report"
	"telegram_bot/internal/modules/report/worker"
	"telegram_bot/internal/modules/submission"
	"telegram_bot/internal/modules/telemetry"
	"telegram_bot/internal/modules/user"
	"telegram_bot/internal/scheduler"
	"telegram_bot/internal/storage"
	"telegram_bot/pkg/postgres"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Config struct {
	Token         string
	PostgresDSN   string
	RedisAddr     string
	RedisPwd      string
	S3Endpoint    string
	S3AccessKey   string
	S3SecretKey   string
	S3Bucket      string
	EncryptionKey string
	PathSalt      string
}

func Run(cfg *Config) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	pgClient, err := postgres.New(cfg.PostgresDSN)
	if err != nil {
		logger.Error("postgres init failed", "err", err)
		os.Exit(1)
	}
	defer pgClient.Close()

	redisClient, err := redis.New(cfg.RedisAddr, cfg.RedisPwd, 0)
	if err != nil {
		logger.Error("redis init failed", "err", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	s3Ctx, s3Cancel := context.WithTimeout(context.Background(), 10*time.Second)
	s3Client, err := s3.New(s3Ctx, cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket)
	s3Cancel()
	if err != nil {
		logger.Error("s3 init failed", "err", err)
		os.Exit(1)
	}

	tgClient, err := telegram.New(cfg.Token)
	if err != nil {
		logger.Error("telegram init failed", "err", err)
		os.Exit(1)
	}

	stateStorage := storage.NewRedisState(redisClient.Client)
	userRepo := user.NewRepo(pgClient)
	subRepo := submission.NewRepo(pgClient)
	cmsRepo := cms.NewRepo(pgClient)

	excelGen := excel.New()
	wordGen := word.New()

	userService := user.NewService(userRepo, tgClient, logger)
	subService := submission.NewService(subRepo, s3Client, tgClient, logger, cfg.EncryptionKey, cfg.PathSalt)
	cmsService := cms.NewService(cmsRepo, logger)

	taskChan := make(chan report.Task, 100)
	reportService := report.NewService(logger, taskChan)

	reportWorkerPool := worker.NewPool(taskChan, userService, excelGen, tgClient, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reportWorkerPool.Start(ctx, 3)

	h := handlers.NewHandler(
		userService,
		subService,
		reportService,
		cmsService,
		stateStorage,
		tgClient.GetBot(),
		logger,
		wordGen,
	)

	latencyOpt := telemetry.NewLatencyOptimizer(logger)
	latencyOpt.StartTelemetrySync(ctx)

	router := bot.NewRouter(tgClient.GetBot(), logger, h, latencyOpt)

	cron := scheduler.New(logger)
	cron.AddDailyTask("GlobalReminders", 10, 0, func(c context.Context) error {
		return userService.SendGlobalReminders(c)
	})
	cron.Start(ctx)

	metricsServer := &http.Server{Addr: ":2112", Handler: promhttp.Handler()}
	go func() {
		logger.Info("Starting Prometheus metrics server on :2112")
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Prometheus metrics server failed", "err", err)
		}
	}()

	go router.Start()

	logger.Info("App running...")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	logger.Info("Shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	reportWorkerPool.Stop()
	metricsServer.Shutdown(shutdownCtx)
	logger.Info("Stop completed")
}
