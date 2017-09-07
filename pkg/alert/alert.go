package alert

import "github.com/zefhemel/ax/pkg/backend/common"

type Alerter interface {
	SendAlert(lm common.LogMessage) error
}
