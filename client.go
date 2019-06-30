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
func (c *Client) FetchStopInfo(stopID string, prognosis bool) (response StopInfo, err error) {
	c.logger.Debugf("fetching info for stop %s", stopID)
	var retry = true

	// 3 times is the maximum possible request due to the following:
	// 1. If we got expired csrf token, we will apply new one ane retry
	// 2. If we didn't found time via prognosis we will try without
	// 3. If we didn't found then, so whatever, drop the tries and return an error
	for i := 0; i < 3 && retry; i++ {
		response, retry, err = c.fetchStopInfo(stopID, prognosis)

		// in case we couldn't found result with prognosis, we will try to find without it
		if prognosis {
			prognosis = !prognosis
		}

		if retry {
			continue
		}

		if err != nil {
			return StopInfo{}, nil
		}
	}

	if err != nil {
		c.logger.Debugf("finished with error: %v", err)
		return StopInfo{}, err
	}

	c.logger.Debugf("finished: %#v", response)
	return response, nil
}

func (c *Client) fetchStopInfo(stopID string, prognosis bool) (stopInfo StopInfo, retry bool, err error) {
	var path, _ = url.Parse(c.host)
	q := path.Query()
	q.Set("csrfToken", c.csrfToken)
	q.Set("locale", c.locale)
	q.Set("id", stopID)
	if prognosis {
		q.Set("mode", "prognosis")
	}
	path.RawQuery = q.Encode()

	c.logger.Debugf("request=%s", path.String())

	req, err := http.NewRequest(http.MethodGet, path.String(), nil)
	if err != nil {
		c.logger.Debugf("error creating request: %v", err)
		return StopInfo{}, false, err
	}

	req.Header.Set("accept-encoding", "gzip")

	resp, err := c.client.Do(req)
	if err != nil {
		return StopInfo{}, false, err
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
		return StopInfo{}, false, err
	}

	if resp.StatusCode == http.StatusNotFound {
		c.logger.Debug("status code is 404 NOT FOUND")
		return StopInfo{}, true, NewWrongStatusCodeError(resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Debugf("status code is %d", resp.StatusCode)
		return StopInfo{}, false, NewWrongStatusCodeError(resp.StatusCode)
	}

	var response StopInfo
	if jerr := json.Unmarshal(respData, &response); jerr != nil {
		c.logger.Debugf("error fetching body: %s with error: %v", respData, jerr)
		return StopInfo{}, false, jerr
	}

	if len(response.CsrfToken) != 0 {
		c.logger.Debugf("found new token, updating to %s", response.CsrfToken)
		c.csrfToken = response.CsrfToken
		c.logger.Debug("fetching info again")
		return StopInfo{}, true, nil
	}

	c.logger.Debugf("fetched: %#+v", response)
	return response, false, nil
}
