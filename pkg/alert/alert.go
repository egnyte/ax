package alert

import "github.com/egnyte/ax/pkg/backend/common"

type Alerter interface {
	SendAlert(lm common.LogMessage) error
}
