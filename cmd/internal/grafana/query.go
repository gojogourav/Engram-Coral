package grafana

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type QueryResult struct {
	Metric string
	Value  float64
}

type PrometheusResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func (c *Client) Query(promQL string) ([]QueryResult, error) {
	endpoint := fmt.Sprintf("%s/api/v1/query", c.BaseURL)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	q := req.URL.Query()
	q.Add("query", promQL)
	req.URL.RawQuery = q.Encode()
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prometheus query failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("prometheus error (status %d): %s", resp.StatusCode, string(body))
	}
	var promResp PrometheusResponse
	if err := json.NewDecoder(resp.Body).Decode(&promResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	var results []QueryResult
	for _, r := range promResp.Data.Result {
		if len(r.Value) != 2 {
			continue
		}
		valueStr, ok := r.Value[1].(string)
		if !ok {
			continue
		}
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue
		}
		metricName := r.Metric["__name__"]
		if metricName == "" {
			metricName = promQL
		}
		results = append(results, QueryResult{Metric: metricName, Value: value})
	}
	return results, nil
}
