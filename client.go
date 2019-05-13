package yandexmapclient

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

const defaultHost = "https://yandex.ru/maps/api/masstransit/getStopInfo"
const defaultLocale = "ru-RU"

type Logger interface {
	Debug(msg string)
	Debugf(format string, args ...interface{})
}

type nopLogger struct{}

func (n *nopLogger) Debug(_ string)                    {}
func (n *nopLogger) Debugf(_ string, _ ...interface{}) {}

type yandexClient struct {
	csrfToken string
	host      string
	locale    string
	client    *http.Client
	logger    Logger
}

// New creates new yandexClient and gets csrfToken to be able to perform requests
func New(opts ...ClientOption) (*yandexClient, error) {
	client := http.DefaultClient
	client.Timeout = time.Second * 15
	jar, _ := cookiejar.New(nil)
	client.Jar = jar

	c := &yandexClient{
		client: client,
		host:   defaultHost,
		locale: defaultLocale,
		logger: &nopLogger{},
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	if len(c.csrfToken) != 0 {
		return c, nil
	}

	if err := c.UpdateToken(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *yandexClient) UpdateToken() error {
	c.logger.Debug("updating token")
	path, _ := url.Parse(c.host)
	q := path.Query()
	q.Set("csrfToken", c.csrfToken)
	path.RawQuery = q.Encode()

	req, _ := http.NewRequest(http.MethodGet, c.host, nil)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer func(cl io.Closer) {
		errClose := cl.Close()
		if errClose != nil {
			c.logger.Debugf("closing resp.Body: %v", errClose)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return NewWrongStatusCodeError(resp.StatusCode)
	}

	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var response = refreshTokenResponse{}
	if err = json.NewDecoder(bytes.NewReader(respData)).Decode(&response); err != nil {
		return err
	}

	if len(response.CsrfToken) == 0 {
		return NewEmptyTokenError()
	}

	c.logger.Debugf("updating csrf token to %s", response.CsrfToken)
	c.csrfToken = response.CsrfToken

	c.client.Jar.SetCookies(resp.Request.URL, resp.Cookies())
	return nil
}

func (c *yandexClient) FetchStopInfo(stopID string) (StopInfo, error) {
	c.logger.Debugf("fetching info for stop %s", stopID)
	response, err := c.FetchStopInfo(stopID)
	if err != nil {
		return StopInfo{}, err
	}

	if response.CsrfToken != nil {
		c.logger.Debugf("found new token, updating to %s", *response.CsrfToken)
		c.csrfToken = *response.CsrfToken
		c.logger.Debug("fetching info again")
		return c.FetchStopInfo(stopID)
	}

	c.logger.Debug("success")
	return response, nil
}

func (c *yandexClient) fetchStopInfo(stopID string) (StopInfo, error) {
	var path, _ = url.Parse(c.host)
	q := path.Query()
	q.Set("csrfToken", c.csrfToken)
	q.Set("locale", c.locale)
	q.Set("id", stopID)
	path.RawQuery = q.Encode()

	req, _ := http.NewRequest(http.MethodGet, c.host, nil)
	resp, err := c.client.Do(req)
	if err != nil {
		return StopInfo{}, err
	}

	defer func(cl io.Closer) { _ = cl.Close() }(resp.Body)
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return StopInfo{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return StopInfo{}, NewWrongStatusCodeError(resp.StatusCode)
	}

	var response StopInfo
	if errDec := json.NewDecoder(bytes.NewReader(respData)).Decode(&response); errDec != nil {
		return StopInfo{}, errDec
	}

	return response, nil
}
