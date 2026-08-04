package ai

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

type cacheEntry struct {
	response string
	expires  time.Time
}

var (
	cacheMu       sync.RWMutex
	cacheStore    = make(map[string]cacheEntry)
	cacheEnabled  = false
	cacheTTL      = 10 * time.Minute
	cacheHitCount int
	cacheMissCount int
)

func SetCacheEnabled(enabled bool) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cacheEnabled = enabled
}

func SetCacheTTL(minutes int) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cacheTTL = time.Duration(minutes) * time.Minute
}

func CacheStats() (hits, misses int) {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	return cacheHitCount, cacheMissCount
}

func ResetCacheStats() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cacheHitCount = 0
	cacheMissCount = 0
}

func ClearCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cacheStore = make(map[string]cacheEntry)
}

func cacheKey(provider, model, systemPrompt, userPrompt string) string {
	h := sha256.New()
	h.Write([]byte(provider))
	h.Write([]byte(":"))
	h.Write([]byte(model))
	h.Write([]byte(":"))
	h.Write([]byte(systemPrompt))
	h.Write([]byte(":"))
	h.Write([]byte(userPrompt))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func cacheGet(key string) (string, bool) {
	cacheMu.RLock()
	defer cacheMu.RUnlock()

	if !cacheEnabled {
		return "", false
	}

	entry, ok := cacheStore[key]
	if !ok {
		cacheMissCount++
		return "", false
	}

	if time.Now().After(entry.expires) {
		delete(cacheStore, key)
		cacheMissCount++
		return "", false
	}

	cacheHitCount++
	return entry.response, true
}

func cacheSet(key, response string) {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	if !cacheEnabled {
		return
	}

	cacheStore[key] = cacheEntry{
		response: response,
		expires:  time.Now().Add(cacheTTL),
	}
}
