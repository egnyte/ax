package kibana

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/zefhemel/ax/pkg/backend/common"
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

func (client *Client) addHeaders(req *http.Request) {
	req.Header.Set("Authorization", client.AuthHeader)
	// TODO: This may seem crazy but this header needs to be set, even if empty
	req.Header.Set("Kbn-Version", "")
	req.Header.Set("Content-Type", "application/x-ldjson")
	req.Header.Set("Accept", "application/json, text/plain, */*")
}

type indexList struct {
	Hits struct {
		Hits []indexListHit `json:"hits"`
	} `json:"hits"`
}

type indexListHit struct {
	Id string `json:"_id"`
}

func (client *Client) ListIndices() ([]string, error) {
	body, err := createMultiSearch(
		JsonObject{
			"query": JsonObject{
				"match_all": JsonObject{},
			},
			"size": 10000,
		},
	)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/es_admin/.kibana/index-pattern/_search?stored_fields=", client.URL), body)
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
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("Authentication failed")
	}
	decoder := json.NewDecoder(resp.Body)
	var data indexList
	err = decoder.Decode(&data)
	if err != nil {
		return nil, err
	}
	// Build list
	indexNames := make([]string, 0, len(data.Hits.Hits))
	for _, indexInfo := range data.Hits.Hits {
		indexNames = append(indexNames, indexInfo.Id)
	}
	return indexNames, nil
}

var _ common.Client = &Client{}
