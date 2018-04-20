package kibana

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/egnyte/ax/pkg/backend/common"
)

type Client struct {
	URL        string
	AuthHeader string
	Index      string
}

func New(url, authHeader, index string) *Client {
	return &Client{
		URL:        url,
		AuthHeader: authHeader,
		Index:      index,
	}
}

func (client *Client) ImplementsAdvancedFilters() bool {
	return false
}

func (client *Client) addHeaders(req *http.Request) {
	req.Header.Set("Authorization", client.AuthHeader)
	// TODO: This may seem crazy but this header needs to be set, even if empty
	req.Header.Set("Kbn-Version", "")
	req.Header.Set("Content-Type", "application/x-ldjson")
}

type indexList struct {
	SavedObjects []struct {
		Type       string `json:"type"`
		Attributes struct {
			Title string `json:"title"`
		}
	} `json:"saved_objects"`
}

type indexListHit struct {
	Id string `json:"_id"`
}

func (client *Client) ListIndices() ([]string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/saved_objects/?type=index-pattern&per_page=10000", client.URL), nil)
	if err != nil {
		return nil, err
	}
	client.addHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("Authentication failed")
	} else if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}
	decoder := json.NewDecoder(resp.Body)
	var data indexList
	err = decoder.Decode(&data)
	if err != nil {
		return nil, err
	}
	// Build list
	indexNames := make([]string, 0, len(data.SavedObjects))
	for _, indexInfo := range data.SavedObjects {
		if indexInfo.Type == "index-pattern" {
			indexNames = append(indexNames, indexInfo.Attributes.Title)
		}
	}
	return indexNames, nil
}

var _ common.Client = &Client{}
