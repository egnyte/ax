package cache

import (
	"testing"
	"time"
)

func TestCacheBehavior(t *testing.T) {
	cache := New("test.cache")
	if cache.Contains("test-key") {
		t.Error("Key should not be present")
	}
	cache.Set("test-key", "value", nil)
	if !cache.Contains("test-key") {
		t.Error("Key should be present")
	}
	if cache.Get("test-key") != "value" {
		t.Errorf("Key has wrong value: %v", cache.Get("test-key"))
	}
	if cache.Flush() != nil {
		t.Errorf("Flush failed")
	}
	cache.Unset("test-key")
	if cache.Contains("test-key") {
		t.Error("Key should not be present")
	}
	cache.Remove()
}

func TestCacheExpiry(t *testing.T) {
	cache := New("test.cache")
	expireDate := time.Now().Add(time.Second)
	cache.Set("test-key", "hello", &expireDate)
	if !cache.Contains("test-key") {
		t.Error("Key not present")
	}
	time.Sleep(time.Second * 2)
	if cache.Contains("test-key") {
		t.Error("Key should have expired")
	}
	// cache.Remove()
}

func TestCachePersistence(t *testing.T) {

}
