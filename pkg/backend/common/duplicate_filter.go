package common

// Skips duplicate messages (based on .ID)
func Dedup(messageChan chan LogMessage) chan LogMessage {
	resultChan := make(chan LogMessage)
	idCache := make(map[string]bool)
	go func() {
		for message := range messageChan {
			if !idCache[message.ID] {
				resultChan <- message
				idCache[message.ID] = true
			}
		}
		close(resultChan)
	}()
	return resultChan
}
