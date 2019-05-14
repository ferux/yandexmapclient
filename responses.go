package yandexmapclient

type refreshTokenResponse struct {
	CsrfToken string `json:"csrfToken"`
}

// StopInfo contains information about incoming transport for stop and csrfToken in case
// server responded with refresh token demand
type StopInfo struct {
	CsrfToken string `json:"csrfToken,omitempty"`
	Error     *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Data *Data `json:"data,omitempty"`
}

// TransportInfo contains departure time and route name
type TransportInfo struct {
	Name          string `json:"name"`
	BriefSchedule struct {
		DepartureTime string `json:"departureTime"`
	} `json:"BriefSchedule"`
}

// Data model
type Data struct {
	Properties Properties `json:"properties"`
}

// Properties model
type Properties struct {
	StopMetaData StopMetaData `json:"StopMetaData"`
}

// StopMetaData model
type StopMetaData struct {
	Transport []TransportInfo `json:"Transport"`
}
