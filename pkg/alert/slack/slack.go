package slack

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/zefhemel/ax/pkg/alert"
	"github.com/zefhemel/ax/pkg/backend/common"
)

type SlackAlerter struct {
	name     string
	token    string
	channel  string
	username string
}

func New(name string, config map[string]string) *SlackAlerter {
	return &SlackAlerter{
		name:     name,
		token:    config["token"],
		channel:  config["channel"],
		username: config["username"],
	}
}

func (alerter *SlackAlerter) SendAlert(lm common.LogMessage) error {
	form := url.Values{}
	form.Add("token", alerter.token)
	form.Add("icon_emoji", ":bell:")
	form.Add("channel", alerter.channel)
	buf, _ := yaml.Marshal(lm.Attributes)
	form.Add("text", fmt.Sprintf("*[%s]* %s", lm.Timestamp.Format(common.TimeFormat), buf))
	form.Add("username", alerter.username)

	url := "https://slack.com/api/chat.postMessage"
	req, err := http.NewRequest("POST", url, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		all, _ := ioutil.ReadAll(res.Body)
		return errors.New(string(all))
	}
	return nil
}

var _ alert.Alerter = &SlackAlerter{}
