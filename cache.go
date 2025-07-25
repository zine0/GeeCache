package geecache

import (
	"github/zine0/GeeCache/lru"
	"sync"
)

type cache struct {
	mu sync.Mutex
	lru *lru.Cache
	cacheBytes uint64
}

func (c *cache) add(key string,value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		c.lru = lru.New(uint64(c.cacheBytes),nil)
	}
	c.lru.Add(key,value)
}

func (c *cache) get(key string) (value ByteView,ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lru == nil {
		return
	}

	if v,ok := c.lru.Get(key);ok {
		return v.(ByteView),ok
	}
	return
}
