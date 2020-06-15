package yandexmapclient

import (
	"encoding/json"
	"strconv"
	"time"
)

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
	CurrentTime  int64        `json:"currentTime"`
}

// StopMetaData model
type StopMetaData struct {
	Transport []TransportInfo `json:"Transport"`
}

// TransportInfo contains departure time and route name
type TransportInfo struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Threads []Thread `json:"threads"`
}

type Thread struct {
	BriefSchedule Brief `json:"BriefSchedule"`
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
	Value string `json:"value"`
}

type TimeInfo struct {
	Time time.Time
}

func (t *TimeInfo) UnmarshalJSON(data []byte) (err error) {
	var ti TimeInfoYandex
	if err = json.Unmarshal(data, &ti); err != nil {
		return err
	}

	unix, err := strconv.ParseInt(ti.Value, 10, 64)
	if err != nil {
		return err
	}

	t.Time = time.Unix(unix, 0)
	return nil
}
