package main

import (
	"context"
	"fmt"
	"time"

	"github.com/egnyte/ax/pkg/alert"
	"github.com/egnyte/ax/pkg/alert/slack"
	"github.com/egnyte/ax/pkg/backend/common"
	"github.com/egnyte/ax/pkg/config"
)

var (
	alertFlags    = addQueryFlags(addAlertCommand)
	alertFlagName string
)

func init() {
	addAlertCommand.Flag("name", "Name for alert").Required().StringVar(&alertFlagName)
}

func setupDest() alert.Alerter {
	return nil
}

func addAlertMain(rc config.RuntimeConfig, client common.Client) {
	alertConfig := config.AlertConfig{
		Env:      rc.ActiveEnv,
		Name:     alertFlagName,
		Selector: *alertFlags,
	}

	fmt.Printf("Config: %+v\n", alertConfig)
	conf := config.LoadConfig()
	conf.Alerts = append(conf.Alerts, alertConfig)
	config.SaveConfig(conf)
}

func watchAlerts(ctx context.Context, rc config.RuntimeConfig, alertConfig config.AlertConfig) {
	var alerter alert.Alerter
	switch alertConfig.Service["backend"] {
	case "slack":
		alerter = slack.New(alertConfig.Name, rc.DataDir, alertConfig.Service)
	default:
		fmt.Printf("Back-end type not supported: %s\n", alertConfig.Service["backend"])
		return
	}
	query := querySelectorsToQuery(&alertConfig.Selector)
	query.Follow = true
	query.MaxResults = 100
	client := determineClient(rc.Config.Environments[alertConfig.Env])
	if client == nil {
		fmt.Println("Cannot obtain a client for", alertConfig)
		return
	}
	fmt.Println("Now waiting for alerts for", alertConfig.Name)
	for message := range client.Query(ctx, query) {
		fmt.Printf("[%s] Sending alert to %s: %+v\n", alertConfig.Name, alertConfig.Service["backend"], message.Map())
		err := alerter.SendAlert(message)
		if err != nil {
			fmt.Println("Couldn't send alert", err)
			continue
		}
	}
}

func alertMain(ctx context.Context, rc config.RuntimeConfig) {
	for _, alert := range rc.Config.Alerts {
		go watchAlerts(ctx, rc, alert)
	}
	for {
		time.Sleep(time.Minute)
	}
}
