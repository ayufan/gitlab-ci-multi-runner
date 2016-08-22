package docker_helpers

import (
	"fmt"
	"net/url"
	"sync"
)

type clientCache struct {
	lock    sync.RWMutex
	clients map[string]Client
}

func (c *clientCache) isCacheable(endpoint string) bool {
	u, err := url.Parse(endpoint)
	if err != nil {
		return false
	}
	return u.Scheme == "unix"
}

func (c *clientCache) fromCache(endpoint string, params ...interface{}) Client {
	if !c.isCacheable(endpoint) {
		return nil
	}

	c.lock.RLock()
	defer c.lock.RUnlock()

	key := endpoint + fmt.Sprintln(params...)
	return c.clients[key]
}

func (c *clientCache) cache(client Client, endpoint string, params ...interface{}) bool {
	if !c.isCacheable(endpoint) {
		return false
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	key := endpoint + fmt.Sprintln(params...)
	c.clients[key] = client
	return true
}
