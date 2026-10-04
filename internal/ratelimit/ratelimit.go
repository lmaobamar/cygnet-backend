package ratelimit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tomasen/realip"

	"github.com/lmaobamar/cygnet-backend/internal/httpx"
)

type Limit struct {
	Max    int64
	Window time.Duration
}

func PerSecond(n int64) Limit { return Limit{Max: n, Window: time.Second} }
func PerMinute(n int64) Limit { return Limit{Max: n, Window: time.Minute} }
func PerHour(n int64) Limit   { return Limit{Max: n, Window: time.Hour} }

type Limiter struct{ rdb *redis.Client }

func New(rdb *redis.Client) *Limiter { return &Limiter{rdb: rdb} }

func (l *Limiter) Check(ctx context.Context, bucket, key string, limit Limit) (bool, time.Duration) {
	sum := sha256.Sum256([]byte(key))
	redisKey := fmt.Sprintf("rl:%s:%s", bucket, hex.EncodeToString(sum[:8]))

	pipe := l.rdb.TxPipeline()
	pipe.SetNX(ctx, redisKey, 0, limit.Window) // starts the window only if one isn't running
	count := pipe.Incr(ctx, redisKey)
	ttl := pipe.PTTL(ctx, redisKey)
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("ratelimit: %v", err)
		return true, 0
	}

	if count.Val() > limit.Max {
		retry := ttl.Val()
		if retry <= 0 {
			retry = limit.Window
		}
		return false, retry
	}
	return true, 0
}

func (l *Limiter) Limit(bucket string, limit Limit) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ok, retry := l.Check(r.Context(), bucket, realip.FromRequest(r), limit)
			if !ok {
				w.Header().Set("Retry-After", strconv.Itoa(int(retry.Seconds())+1))
				httpx.WriteJson(w, http.StatusTooManyRequests, httpx.ErrorResponse{
					Code: httpx.RateLimitedError, Message: "too many requests",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
