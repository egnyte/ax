package slack

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/egnyte/ax/pkg/alert"
	"github.com/egnyte/ax/pkg/backend/common"
	"github.com/egnyte/ax/pkg/cache"
)

type SlackAlerter struct {
	name      string
	token     string
	channel   string
	username  string
	iconEmoji string
	seenCache *cache.Cache
}

func New(name, dataDir string, config map[string]string) *SlackAlerter {
	return &SlackAlerter{
		name:      name,
		token:     config["token"],
		channel:   config["channel"],
		username:  config["username"],
		iconEmoji: config["icon_emoji"],
		seenCache: cache.New(fmt.Sprintf("%s/alert-%s-seen.json", dataDir, name)),
	}
}

type slackResponse struct {
	Ok      bool
	Error   string
	Channel string
	Ts      string
}

func slackPostRequest(url string, form url.Values) (*slackResponse, error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(res.Body)
	var slackResponse slackResponse
	err = decoder.Decode(&slackResponse)
	if err != nil {
		return nil, err
	}
	if !slackResponse.Ok {
		return nil, errors.New(slackResponse.Error)
	}
	return &slackResponse, nil
}

func (alerter *SlackAlerter) SendAlert(lm common.LogMessage) error {
	contentHash := lm.ContentHash()
	buf, _ := yaml.Marshal(lm.Attributes)
	if alerter.seenCache.Contains(contentHash) {
		fmt.Println("Skipping", lm)
		/*
			ts := alerter.seenCache.GetString(contentHash)
			// TODO: Add reaction
			form := url.Values{}
			form.Add("token", alerter.token)
			form.Add("channel", alerter.channel)
			form.Add("ts", ts)
			form.Add("text", fmt.Sprintf("(:exclamation: %d) *[%s]* %s", occurrences, lm.Timestamp.Format(common.TimeFormat), buf))
			form.Add("username", alerter.username)
			fmt.Println("Skipped", lm)
			resp, err := slackPostRequest("https://slack.com/api/chat.update", form)
			fmt.Println("Response from slack", resp, err)
			return err
		*/
		return nil
	}
	form := url.Values{}
	form.Add("token", alerter.token)
	form.Add("icon_emoji", alerter.iconEmoji)
	form.Add("channel", alerter.channel)
	form.Add("text", fmt.Sprintf("*[%s]* %s", lm.Timestamp.Format(common.TimeFormat), buf))
	form.Add("username", alerter.username)
	resp, err := slackPostRequest("https://slack.com/api/chat.postMessage", form)
	if err != nil {
		return err
	}
	if !resp.Ok {
		return errors.New(resp.Error)
	}

	expire := time.Now().Add(time.Hour * 24 * 7)
	alerter.seenCache.Set(contentHash, resp.Ts, &expire)

	err = alerter.seenCache.Flush()
	if err != nil {
		return err
	}
	return nil
}

var _ alert.Alerter = &SlackAlerter{}
