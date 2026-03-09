package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "bot_errors_total",
		Help: "Общее количество ошибок",
	}, []string{"type"})

	UpdatesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "bot_updates_total",
		Help: "Счетчик всех входящих апдейтов",
	}, []string{"action"}) // action: "/start", "Сдать ДЗ", "photo"

	SubmissionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "bot_submissions_total",
		Help: "Счетчик сданных работ",
	}, []string{"type"}) // type: "homework", "notes"

	S3OperationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "bot_s3_operations_total",
		Help: "Счетчик обращений к S3",
	}, []string{"operation", "status"}) // operation: "upload", "download" | status: "success", "error"

	S3OperationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "bot_s3_duration_seconds",
		Help:    "Время выполнения запросов к S3",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation"})

	DBQueriesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "bot_db_queries_total",
		Help: "Счетчик запросов к базе данных",
	}, []string{"query_name", "status"}) // query_name: "get_user", "insert_submission"

	DBQueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "bot_db_duration_seconds",
		Help:    "Время выполнения запросов к БД",
		Buckets: prometheus.DefBuckets,
	}, []string{"query_name"})
)
