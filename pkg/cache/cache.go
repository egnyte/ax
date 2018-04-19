package cache

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

type cacheItem struct {
	Value      interface{} `json:"value"`
	ExpireDate *time.Time  `json:"expire"`
}

type Cache struct {
	path string
	data map[string]cacheItem
}

func New(path string) *Cache {
	cache := &Cache{
		path: path,
		data: make(map[string]cacheItem),
	}
	file, err := os.Open(cache.path)
	if err == nil {
		defer file.Close()
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&cache.data)
		if err != nil {
			log.Printf("Error decoding cache: %v\n", err)
			// Let's recover gracefully
			cache.data = make(map[string]cacheItem)
			cache.Flush()
		}
	}
	return cache
}

func (cache *Cache) Contains(key string) bool {
	if value, ok := cache.data[key]; ok {
		if value.ExpireDate != nil && value.ExpireDate.Before(time.Now()) {
			delete(cache.data, key)
			return false
		}
		return true
	}
	return false
}

func (cache *Cache) Set(key string, value interface{}, expire *time.Time) {
	cache.data[key] = cacheItem{value, expire}
}

func (cache *Cache) Unset(key string) {
	delete(cache.data, key)
}

func (cache *Cache) Get(key string) interface{} {
	// Could be optimized slightly, but oh well
	if cache.Contains(key) {
		return cache.data[key].Value
	} else {
		return nil
	}
}

func (cache *Cache) GetString(key string) string {
	return cache.Get(key).(string)
}

func (cache *Cache) GetMap(key string) map[string]interface{} {
	return cache.Get(key).(map[string]interface{})
}

func (cache *Cache) Flush() error {
	file, err := os.OpenFile(cache.path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	file.Truncate(0)
	defer file.Close()
	encoder := json.NewEncoder(file)
	return encoder.Encode(cache.data)
}

func (cache *Cache) Remove() error {
	return os.Remove(cache.path)
}
