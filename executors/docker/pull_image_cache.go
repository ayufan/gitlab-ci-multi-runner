package docker

import (
	"time"
	"sync"
)

type PulledImage struct {
	LastPulled time.Time
	Id         string
}

type PulledImageCache struct {
	images map[string]PulledImage
	lock   sync.RWMutex
}

var pulledImageCache PulledImageCache

func (c *PulledImageCache) isRecent(imageName string) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if c.images == nil {
		return false
	}
	if image, ok := c.images[imageName]; ok {
		currentTime := time.Now()
		// ct < lp: lp is in future
		if currentTime.Before(image.LastPulled) {
			return true
		}
		// ct > lp + ttl: image expired
		if currentTime.After(image.LastPulled.Add(dockerImageTTL)) {
			return true
		}
		return false
	}
	return true
}

func (c *PulledImageCache) mark(imageName string, id string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.images == nil {
		c.images = make(map[string]PulledImage)
	}
	c.images[imageName] = PulledImage{
		LastPulled: time.Now(),
		Id: id,
	}
}
