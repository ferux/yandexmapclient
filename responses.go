package yandexmapclient

type refreshTokenResponse struct {
	CsrfToken string `json:"csrfToken"`
}

type StopInfo struct {
	CsrfToken *string `json:"csrfToken"`
	Error     *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Data struct {
		Properties struct {
			StopMetaData struct {
				Transport []TransportInfo `json:"Transport"`
			} `json:"StopMetaData"`
		} `json:"properties"`
	} `json:"data"`
}

type TransportInfo struct {
	Name          string `json:"name"`
	BriefSChedule struct {
		DepartureTime string `json:"departureTime"`
	} `json:"BriefSchedule"`
}
