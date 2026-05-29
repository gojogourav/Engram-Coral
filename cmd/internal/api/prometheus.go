package api

import (
	"encoding/json"
	"net/http"
)

// PrometheusPayload matches the standard Alertmanager JSON webhook schema
type PrometheusPayload struct {
	Status string `json:"status"`
	Alerts []struct {
		Labels struct {
			AlertName string `json:"alertname"`
			Pod       string `json:"pod"`
			Namespace string `json:"namespace"`
		} `json:"labels"`
		Annotations struct {
			Description string `json:"description"`
		} `json:"annotations"`
	} `json:"alerts"`
}

func (g *Gateway) PrometheusAlertHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload PrometheusPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Bad request: invalid json", http.StatusBadRequest)
		return
	}

	for _, alert := range payload.Alerts {
		if payload.Status == "firing" {
			go g.TriggerSelfHealing(
				alert.Labels.AlertName,
				alert.Labels.Pod,
				alert.Labels.Namespace,
				alert.Annotations.Description,
			)
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"processing"}`))
}
