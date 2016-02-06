package docker

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestImageNotFound(t *testing.T) {
	cache := PulledImageCache{}
	result := cache.isExpired("test")
	assert.Equal(t, true, result)
}

func TestMarkedImage(t *testing.T) {
	cache := PulledImageCache{}
	cache.mark("test", "id", time.Minute)
	result := cache.isExpired("test")
	assert.Equal(t, false, result)
}

func TestImageTimeout(t *testing.T) {
	cache := PulledImageCache{}
	cache.mark("test", "id", 0*time.Second)
	result := cache.isExpired("test")
	assert.Equal(t, true, result)
}

func TestImagePulledInTheFuture(t *testing.T) {
	cache := PulledImageCache{}
	cache.images = make(map[string]PulledImage)
	cache.images["test"] = PulledImage{
		ID:         "id",
		LastPulled: time.Now().Add(time.Hour),
		TTL:        time.Second,
	}
	result := cache.isExpired("test")
	assert.Equal(t, true, result)
}
