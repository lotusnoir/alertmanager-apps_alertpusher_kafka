package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/optiopay/kafka/v2/proto"
	"github.com/vma/glog"
)

type KafkaAlert struct {
	Name   string `json:"name"`
	Origin string `json:"origin"`
	Topic  string `json:"topic"`
	Data   Data   `json:"data"`
}

type Data struct {
	Fingerprint  string     `json:"fingerprint"`
	Status       string     `json:"status"`
	Severity     string     `json:"severity"`
	Subject      string     `json:"subject"`
	Description  string     `json:"description"`
	Destination  []string   `json:"destination"`
	CreateTicket bool       `json:"create_ticket"`
	StartTime    time.Time  `json:"started_at"`
	EndTime      *time.Time `json:"ended_at,omitempty"`
	Raw          Alert      `json:"raw"`
}

const DateTimeFormat = "2006-01-02 15:04:05-07"

var nraPat = regexp.MustCompile(`.+\.([^.]+)\.[^.]+\.example\.net`)

func kafkaExport(alerts []Alert) {
	for _, a := range alerts {
		if a.Status == "resolved" && *skipResolved {
			glog.V(1).Info("skipping resolved alert")
			continue
		}
		if a.Labels.Email == "" || a.Labels.Severity == "ok" {
			glog.V(1).Info("skipping alert with empty email or severity OK")
			continue
		}

		if a.Labels.Instance == "" {
			if m := nraPat.FindStringSubmatch(a.Labels.HostName); len(m) >= 2 {
				a.Labels.Instance = m[1]
			}
		}
		ka := KafkaAlert{
			Name:   a.Labels.AlertName,
			Origin: "Horus",
			Topic:  *kafkaTopic,
			Data: Data{
				Fingerprint:  a.Fingerprint,
				Status:       a.Status,
				Severity:     a.Labels.Severity,
				Destination:  strings.Split(strings.ReplaceAll(a.Labels.Email, " ", ""), ","),
				CreateTicket: a.Labels.Jira == "true",
				StartTime:    a.StartsAt,
				Raw:          a,
			},
		}
		status := strings.ToUpper(a.Labels.Severity)
		if a.Status == "resolved" {
			status = "OK"
		}
		ka.Data.Subject = fmt.Sprintf("[ALRT][%s] %s: %s", status, a.Labels.Instance, a.Annotations.Value)

		var desc []string
		desc = append(desc, fmt.Sprintf("status=%s", a.Status))
		desc = append(desc, fmt.Sprintf("output=%s", a.Annotations.Value))
		desc = append(desc, fmt.Sprintf("host=%s", a.Labels.HostName))
		desc = append(desc, fmt.Sprintf("start_time=%s", a.StartsAt.Local().Format(DateTimeFormat)))
		if !a.EndsAt.IsZero() {
			ka.Data.EndTime = &a.EndsAt
			desc = append(desc, fmt.Sprintf("end_time=%s", a.EndsAt.Local().Format(DateTimeFormat)))
		}
		if a.Labels.Uplink != "" {
			desc = append(desc, fmt.Sprintf("uplink=%s", a.Labels.Uplink))
		}
		if a.Labels.Component != "" {
			desc = append(desc, fmt.Sprintf("component=%s", a.Labels.Component))
		}
		ka.Data.Description = strings.Join(desc, "\n")

		b, err := json.Marshal(ka)
		if err != nil {
			glog.Errorf("marshal kafka alert: %v", err)
			continue
		}
		glog.Infof(">> kafka alert:\n%s\n", b)
		msg := &proto.Message{Value: b}
		if _, err := producer.Produce(*kafkaTopic, *kafkaPart, msg); err != nil {
			glog.Errorf("kafka write: %v", err)
		}
	}
}
