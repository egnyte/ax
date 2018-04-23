package complete

import (
	"fmt"
	"time"

	"log"

	"github.com/egnyte/ax/pkg/backend/common"
	"github.com/egnyte/ax/pkg/cache"
	"github.com/egnyte/ax/pkg/config"
)

const cacheFilename = "attribute-cache.json"

func GatherCompletionInfo(rc config.RuntimeConfig, messages <-chan common.LogMessage) <-chan common.LogMessage {
	cache := cache.New(fmt.Sprintf("%s/%s", rc.DataDir, cacheFilename))
	completionsKey := fmt.Sprintf("completions:%s", rc.ActiveEnv)
	attrNames := make(map[string]bool)

	// This will be read back as a map[string]interface{} not a bool
	if existingAttributes, ok := cache.Get(completionsKey).(map[string]interface{}); ok {
		log.Println("Found existing attribute cache")
		for existingAttr := range existingAttributes {
			attrNames[existingAttr] = true
		}
	}
	changed := true
	stopFlushing := make(chan struct{})
	resultChan := make(chan common.LogMessage)
	go func() {
		for message := range messages {
			resultChan <- message
			for k := range message.Attributes {
				if !attrNames[k] {
					attrNames[k] = true
					changed = true
				}
			}
		}
		close(resultChan)
		stopFlushing <- struct{}{}
	}()
	go func() {
		// Flushes cache to disk every 5 seconds until completed
		for {
			shouldBreak := false
			select {
			case <-stopFlushing:
				shouldBreak = true
			case <-time.After(5 * time.Second):
			}
			if changed {
				log.Println("Flushing cache")
				cache.Set(completionsKey, attrNames, nil)
				err := cache.Flush()
				if err != nil {
					log.Println("Could not flush cache:", err)
				}
				changed = false
			}
			if shouldBreak {
				break
			}
		}
		log.Println("Stopped flush loop")
	}()
	return resultChan
}

func GetCompletions(rc config.RuntimeConfig) map[string]bool {
	cache := cache.New(fmt.Sprintf("%s/%s", rc.DataDir, cacheFilename))
	res := cache.Get(fmt.Sprintf("completions:%s", rc.ActiveEnv))
	result := make(map[string]bool)
	if attrNames, ok := res.(map[string]interface{}); ok {
		for attrName := range attrNames {
			result[attrName] = true
		}
	}
	return result
}
