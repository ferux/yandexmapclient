package yandexmapclient

import (
	"encoding/json"
	"time"
)

const jsonTimeFormat = "Mon Jan 02 2006 15:04:05 GMT-0700 (MST)"

type refreshTokenResponse struct {
	CsrfToken string `json:"csrfToken"`
}

// StopInfo contains information about incoming transport for stop and csrfToken in case
// server responded with refresh token demand
type StopInfo struct {
	CsrfToken string          `json:"csrfToken,omitempty"`
	Error     *YandexMapError `json:"error,omitempty"`
	Data      *Data           `json:"data,omitempty"`
}

// Data model
type Data struct {
	Properties Properties `json:"properties"`
}

// Properties model
type Properties struct {
	StopMetaData StopMetaData `json:"StopMetaData"`
	CurrentTime  time.Time    `json:"currentTime"`
}

func (p *Properties) UnmarshalJSON(data []byte) (err error) {
	var in struct {
		StopMetaData StopMetaData `json:"StopMetaData"`
		CurrentTime  string       `json:"currentTime"`
	}

	if err = json.Unmarshal(data, &in); err != nil {
		return err
	}

	t, err := time.Parse(jsonTimeFormat, in.CurrentTime)
	if err != nil {
		return err
	}

	p.StopMetaData = in.StopMetaData
	p.CurrentTime = t
	return nil
}

// StopMetaData model
type StopMetaData struct {
	Transport []TransportInfo `json:"Transport"`
}

// TransportInfo contains departure time and route name
type TransportInfo struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	BriefSchedule Brief  `json:"BriefSchedule"`
}

// Brief contains unit's schedule info. It may or may not have DepartureTime
type Brief struct {
	DepartureTime *string   `json:"departureTime"`
	Events        []Events  `json:"Events"`
	Frequency     Frequency `json:"Frequency"`
}

type Events struct {
	Scheduled TimeInfo `json:"scheduled"`
	Estimated TimeInfo `json:"estimated"`
}

type Frequency struct {
	Value int64    `json:"value"`
	Begin TimeInfo `json:"begin"`
	End   TimeInfo `json:"end"`
}

type TimeInfoYandex struct {
	Value int64 `json:"value"`
}

type TimeInfo struct {
	Time time.Time
}

func (t *TimeInfo) UnmarshalJSON(data []byte) (err error) {
	var ti TimeInfoYandex
	if err = json.Unmarshal(data, &ti); err != nil {
		return err
	}

	t.Time = time.Unix(ti.Value, 0)
	return nil
}
