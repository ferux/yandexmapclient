package yandexmapclient

import (
	"bytes"
	"compress/gzip"
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

// Logger interface for logging some things
type Logger interface {
	Debug(msg string)
	Debugf(format string, args ...interface{})
}

type nopLogger struct{}

func (nopLogger) Debug(_ string)                    {}
func (nopLogger) Debugf(_ string, _ ...interface{}) {}

// Client interacts with yandex maps API
type Client struct {
	csrfToken string
	host      string
	locale    string
	client    *http.Client
	logger    Logger
}

// New creates new yandexClient and gets csrfToken to be able to perform requests
func New(opts ...ClientOption) (*Client, error) {
	client := http.DefaultClient
	client.Timeout = time.Second * 15
	jar, _ := cookiejar.New(nil)
	client.Jar = jar

	c := &Client{
		client: client,
		host:   defaultHost,
		locale: defaultLocale,
		logger: nopLogger{},
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

// UpdateToken gets new csrfToken if needed
func (c *Client) UpdateToken() error {
	c.logger.Debug("updating token")
	path, _ := url.Parse(c.host)
	q := path.Query()
	q.Set("csrfToken", c.csrfToken)
	path.RawQuery = q.Encode()

	c.logger.Debugf("request=%s", path.String())

	req, _ := http.NewRequest(http.MethodGet, path.String(), nil)
	req.Header.Set("accept-encoding", "gzip,deflate,br")

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
	c.client.Jar.SetCookies(path, resp.Cookies())

	return nil
}

// FetchStopInfo gets stop info by specific stop id. If first request gets `invalid csrf token` response it applies it and retries
func (c *Client) FetchStopInfo(stopID string) (StopInfo, error) {
	c.logger.Debugf("fetching info for stop %s", stopID)
	response, err := c.fetchStopInfo(stopID)
	if err != nil {
		return StopInfo{}, err
	}

	if len(response.CsrfToken) != 0 {
		c.logger.Debugf("found new token, updating to %s", response.CsrfToken)
		c.csrfToken = response.CsrfToken
		c.logger.Debug("fetching info again")
		return c.fetchStopInfo(stopID)
	}

	c.logger.Debug("success")
	return response, nil
}

func (c *Client) fetchStopInfo(stopID string) (StopInfo, error) {
	var path, _ = url.Parse(c.host)
	q := path.Query()
	q.Set("csrfToken", c.csrfToken)
	q.Set("locale", c.locale)
	q.Set("id", stopID)
	path.RawQuery = q.Encode()

	c.logger.Debugf("request=%s", path.String())

	req, _ := http.NewRequest(http.MethodGet, path.String(), nil)
	req.Header.Set("accept-encoding", "gzip")

	resp, err := c.client.Do(req)
	if err != nil {
		return StopInfo{}, err
	}
	defer func(cl io.Closer) {
		errClose := cl.Close()
		if errClose != nil {
			c.logger.Debugf("closing resp.Body: %v", errClose)
		}
	}(resp.Body)
	var reader io.ReadCloser
	switch resp.Header.Get("content-encoding") {
	case "gzip":
		reader, _ = gzip.NewReader(resp.Body)
	default:
		reader = resp.Body
	}

	respData, err := ioutil.ReadAll(reader)
	if err != nil {
		return StopInfo{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return StopInfo{}, NewWrongStatusCodeError(resp.StatusCode)
	}

	var response StopInfo
	if errDec := json.NewDecoder(bytes.NewReader(respData)).Decode(&response); errDec != nil {
		c.logger.Debugf("error fetching body: %s", respData)
		return StopInfo{}, errDec
	}

	return response, nil
}
