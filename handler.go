package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/vma/glog"
)

type PromAlerts struct {
	Alerts []Alert `json:"alerts"`
}

type Alert struct {
	Status      string      `json:"status"`
	Labels      Labels      `json:"labels"`
	Annotations Annotations `json:"annotations"`
	StartsAt    time.Time   `json:"startsAt"`
	EndsAt      time.Time   `json:"endsAt"`
	Fingerprint string      `json:"fingerprint"`
}

type Labels struct {
	AlertName string `json:"alertname"`
	HostIndex string `json:"host_index"`
	HostName  string `json:"host_name"`
	Instance  string `json:"instance"`
	Uplink    string `json:"uplink"`
	Category  string `json:"category"`
	Vendor    string `json:"vendor"`
	Model     string `json:"model"`
	Severity  string `json:"severity"`
	Component string `json:"component"`
	Service   string `json:"service"`
	Email     string `json:"email"`
	Jira      string `json:"jira"`
}

type Annotations struct {
	Description string `json:"description,omitempty"`
	Value       string `json:"value"`
}

func HandleAlert(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		glog.Errorf("read body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	r.Body.Close()
	glog.V(2).Infof(">>\n%s\n", b)

	var a PromAlerts
	if err := json.Unmarshal(b, &a); err != nil {
		glog.Errorf("decode prom alert: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	glog.V(2).Infof("prom alerts: %+v", a)
	kafkaExport(a.Alerts)
	w.WriteHeader(http.StatusAccepted)
	w.Header().Set("Connection", "close")
	return
}
