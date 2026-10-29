package handler

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"leadflow/internal/platform/queue"
)

type HealthHandler struct {
	db         *pgxpool.Pool
	redisQueue *queue.RedisQueue
}

func NewHealthHandler(db *pgxpool.Pool, redisQueue *queue.RedisQueue) *HealthHandler {
	return &HealthHandler{
		db:         db,
		redisQueue: redisQueue,
	}
}

func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r, 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, `{"status":"not_ready", "dependency":"postgres"}`)
		return
	}

	if err := h.redisQueue.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, `{"status":"not_ready", "dependency":"redis"}`)
		return
	}

	writeJSON(w, http.StatusOK, `{"status": "ready"}`)
}
