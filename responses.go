package yandexmapclient

type refreshTokenResponse struct {
	CsrfToken string `json:"csrfToken"`
}

type StopInfo struct {
	CsrfToken string `json:"csrfToken,omitempty"`
	Error     *struct {
		Code    int    `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	} `json:"error,omitempty"`
	Data *Data `json:"data,omitempty"`
}

type TransportInfo struct {
	Name          string `json:"name,omitempty"`
	BriefSchedule struct {
		DepartureTime string `json:"departureTime,omitempty"`
	} `json:"BriefSchedule,omitempty"`
}

type Data struct {
	Properties Properties `json:"properties,omitempty"`
}

type Properties struct {
	StopMetaData StopMetaData `json:"StopMetaData,omitempty"`
}

type StopMetaData struct {
	Transport []TransportInfo `json:"Transport,omitempty"`
}
