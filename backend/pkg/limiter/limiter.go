// Package limiter предоставляет middleware для ограничения частоты запросов (rate limiting)
// с использованием алгоритма token bucket. Поддерживает автоматическую очистку старых посетителей
// и глобальный реестр для graceful shutdown.
package limiter

import (
	"context"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// visitor хранит лимитер и время последнего визита для одного IP.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen int64 // атомарное значение в наносекундах
}

// rateLimiter — внутренняя структура, управляющая картой посетителей.
type rateLimiter struct {
	mu       sync.RWMutex
	visitors map[string]*visitor
	limit    rate.Limit
	burst    int
	ttl      time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
}

// newRateLimiter создаёт новый экземпляр rateLimiter.
func newRateLimiter(rps, burst int, ttl time.Duration) *rateLimiter {
	ctx, cancel := context.WithCancel(context.Background())
	return &rateLimiter{
		visitors: make(map[string]*visitor),
		limit:    rate.Limit(rps),
		burst:    burst,
		ttl:      ttl,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// getVisitor возвращает лимитер для заданного IP, создавая его при необходимости.
// Безопасен для конкурентного использования.
func (l *rateLimiter) getVisitor(ip string) *rate.Limiter {
	// Быстрый путь: читаем под RLock
	l.mu.RLock()
	v, exists := l.visitors[ip]
	l.mu.RUnlock()
	if exists {
		atomic.StoreInt64(&v.lastSeen, time.Now().UnixNano())
		return v.limiter
	}

	// Медленный путь: создаём нового посетителя
	limiter := rate.NewLimiter(l.limit, l.burst)
	newVisitor := &visitor{
		limiter:  limiter,
		lastSeen: time.Now().UnixNano(),
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	// Double-check: возможно, другой поток уже создал запись
	if v, exists := l.visitors[ip]; exists {
		atomic.StoreInt64(&v.lastSeen, time.Now().UnixNano())
		return v.limiter
	}
	l.visitors[ip] = newVisitor
	return limiter
}

// cleanupVisitors запускает фоновую очистку старых записей раз в минуту.
func (l *rateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now().UnixNano()
			ttlNs := l.ttl.Nanoseconds()

			l.mu.Lock()
			// Собираем ключи для удаления (короткая блокировка)
			toDelete := make([]string, 0, len(l.visitors)/10)
			for ip, v := range l.visitors {
				if now-atomic.LoadInt64(&v.lastSeen) > ttlNs {
					toDelete = append(toDelete, ip)
				}
			}
			// Удаляем
			for _, ip := range toDelete {
				delete(l.visitors, ip)
			}
			l.mu.Unlock()

		case <-l.ctx.Done():
			return
		}
	}
}

// Stop останавливает фоновую горутину очистки для данного лимитера.
func (l *rateLimiter) Stop() {
	l.cancel()
}

// middleware — внутренний обработчик для Gin.
func (l *rateLimiter) middleware(c *gin.Context) {
	// Надёжное определение IP клиента
	ip := c.ClientIP()
	if ip == "" {
		host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err != nil {
			ip = c.Request.RemoteAddr
		} else {
			ip = host
		}
	}

	if !l.getVisitor(ip).Allow() {
		c.Header("Retry-After", "60")
		c.AbortWithStatus(http.StatusTooManyRequests)
		return
	}
	c.Next()
}

// ------------------------------
// Глобальный реестр всех лимитеров
// ------------------------------

var (
	allLimiters []*rateLimiter
	muAll       sync.Mutex
)

// Limit создаёт новый rate limiter middleware, регистрирует его в глобальном реестре
// и возвращает gin.HandlerFunc для использования в роутерах.
//
// Параметры:
//   - rps:  максимальное количество запросов в секунду (rate.Limit)
//   - burst: максимальный размер burst'а (токены)
//   - ttl:   время жизни записи об IP после последнего запроса
//
// Пример:
//
//	router.GET("/api", limiter.Limit(10, 5, time.Minute), handler)
func Limit(rps, burst int, ttl time.Duration) gin.HandlerFunc {
	l := newRateLimiter(rps, burst, ttl)

	muAll.Lock()
	allLimiters = append(allLimiters, l)
	muAll.Unlock()

	go l.cleanupVisitors()
	return l.middleware
}

// StopAll останавливает фоновые горутины очистки для всех созданных лимитеров.
// Рекомендуется вызывать при graceful shutdown сервера.
func StopAll() {
	muAll.Lock()
	defer muAll.Unlock()
	for _, l := range allLimiters {
		l.Stop()
	}
	allLimiters = nil
}
