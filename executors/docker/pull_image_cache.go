package docker

import (
	"sync"
	"time"
)

type PulledImage struct {
	ID         string
	LastPulled time.Time
	TTL        time.Duration
}

type PulledImageCache struct {
	images map[string]PulledImage
	lock   sync.RWMutex
}

var pulledImageCache PulledImageCache

func (c *PulledImageCache) isExpired(imageName string) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if c.images == nil {
		return true
	}
	if image, ok := c.images[imageName]; ok {
		currentTime := time.Now()
		// ct < lp: lp is in future
		if currentTime.Before(image.LastPulled) {
			return true
		}
		// ct > lp + ttl: image expired
		if currentTime.After(image.LastPulled.Add(image.TTL)) {
			return true
		}
		return false
	}
	return true
}

func (c *PulledImageCache) mark(imageName string, id string, ttl time.Duration) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.images == nil {
		c.images = make(map[string]PulledImage)
	}
	c.images[imageName] = PulledImage{
		LastPulled: time.Now(),
		ID:         id,
		TTL:        ttl,
	}
}
