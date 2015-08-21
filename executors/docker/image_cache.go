package docker

import (
	"time"
	"sync"
)

type DownloadedImage struct {
	LastPulled time.Time
}

var downloadedImages map[string]DownloadedImage
var downloadedImagesLock sync.RWMutex

func shouldDownloadImage(id string) bool {
	downloadedImagesLock.RLock()
	defer downloadedImagesLock.RUnlock()
	if downloadedImage, ok := downloadedImages[id]; ok {
		currentTime := time.Now()
		// ct < lp: lp is in future
		if currentTime.Before(downloadedImage.LastPulled) {
			return true
		}
		// ct > lp + ttl: image expired
		if currentTime.After(downloadedImage.LastPulled.Add(dockerImageTTL)) {
			return true
		}
		return false
	}
	return true
}

func markAsDownloaded(id string) {
	downloadedImagesLock.Lock()
	defer downloadedImagesLock.Unlock()
	downloadedImages[id] = DownloadedImage{
		LastPulled: time.Now(),
	}
}

func init() {
	downloadedImages = make(map[string]DownloadedImage)
}
