package yandexmapclient

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

const defaultHost = "https://yandex.ru/maps/api/masstransit/getStopInfo"
const defaultLocale = "ru_RU"
const defaultLang = "ru"
const defaultPoolSize = 1 << 20 // 1 MiB

// Logger interface for logging some things
type Logger interface {
	Debug(msg string)
	Debugf(format string, args ...interface{})
}

type nopLogger struct{}

func (nopLogger) Debug(_ string)                    {}
func (nopLogger) Debugf(_ string, _ ...interface{}) {}
func (nopLogger) Module(_ string) ModuleLogger      { return nopLogger{} }

// Client interacts with yandex maps API
type Client struct {
	csrfToken   string
	host        string
	locale      string
	lang        string
	poolMaxSize int
	client      *http.Client
	logger      ModuleLogger
	pool        *bytesPool
}

// New creates new yandexClient and gets csrfToken to be able to perform requests
func New(opts ...ClientOption) (*Client, error) {
	client := http.DefaultClient
	client.Timeout = time.Second * 15
	jar, _ := cookiejar.New(nil)
	client.Jar = jar

	c := &Client{
		client:      client,
		host:        defaultHost,
		locale:      defaultLocale,
		lang:        defaultLang,
		poolMaxSize: defaultPoolSize,
		logger:      &nopLogger{},
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	c.pool = newCachedPool(c.poolMaxSize, c.logger)

	if len(c.csrfToken) != 0 {
		c.logger.Debug("csrf_token found")
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
	path, err := url.Parse(c.host)
	if err != nil {
		return fmt.Errorf("parsing url: %w", err)
	}

	q := path.Query()
	q.Set("csrfToken", c.csrfToken)
	path.RawQuery = q.Encode()

	c.logger.Debugf("request=%s", path.String())

	req, _ := http.NewRequest(http.MethodGet, path.String(), nil)
	req.Header.Set("accept-encoding", "gzip,deflate,br")
	req.Header.Set("accept", "*/*")
	req.Header.Set("cache-control", "no-cache")

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

	var n int64
	var response = refreshTokenResponse{}
	c.pool.Apply(func(buf *bytes.Buffer) {
		n, err = io.Copy(buf, resp.Body)
		if err != nil {
			err = fmt.Errorf("copying data from body: %w", err)

			return
		}

		if n == 0 {
			return
		}

		if resp.StatusCode != http.StatusOK {
			c.logger.Debugf("error data: %s", buf.String())

			err = NewWrongStatusCodeError(resp.StatusCode)
		}

		err = json.NewDecoder(buf).Decode(&response)
		if err != nil {
			err = fmt.Errorf("decoding data: %w", err)
		}
	})
	if err != nil {
		return err
	}

	if len(response.CsrfToken) == 0 {
		return NewEmptyTokenError()
	}

	c.logger.Debugf("updating csrf token to %s", response.CsrfToken)

	for _, cookie := range resp.Cookies() {
		c.logger.Debugf("raw cookie: %s", cookie.Raw)
	}

	c.csrfToken = response.CsrfToken
	c.client.Jar.SetCookies(path, resp.Cookies())

	return nil
}

// FetchStopInfo gets stop info by specific stop id. If first request gets `invalid csrf token` response it applies it and retries
func (c *Client) FetchStopInfo(ctx context.Context, stopID string, prognosis bool) (response StopInfo, err error) {
	c.logger.Debugf("fetching info for stop %s", stopID)
	u, _ := url.Parse(c.host)
	c.logger.Debug("cookies: " + c.host)

	for _, cookie := range c.client.Jar.Cookies(u) {
		c.logger.Debug(cookie.String())
	}

	var retry = true

	// 3 times is the maximum possible request due to the following:
	// 1. If we got expired csrf token, we will apply new one ane retry
	// 2. If we didn't found time via prognosis we will try without
	// 3. If we didn't found then, so whatever, drop the tries and return an error
	for i := 0; i < 3 && retry; i++ {
		response, retry, err = c.fetchStopInfo(ctx, stopID, prognosis)

		// in case we couldn't found result with prognosis, we will try to find without it
		if prognosis {
			prognosis = !prognosis
		}

		if retry {
			continue
		}

		if err != nil {
			return StopInfo{}, err
		}
	}

	if err != nil {
		c.logger.Debugf("finished with error: %v", err)
		return StopInfo{}, err
	}

	c.logger.Debugf("finished: %#v", response)
	return response, nil
}

func (c *Client) fetchStopInfo(ctx context.Context, stopID string, prognosis bool) (stopInfo StopInfo, retry bool, err error) {
	var path *url.URL
	path, err = url.Parse(c.host)
	if err != nil {
		return StopInfo{}, false, fmt.Errorf("parsing host: %w", err)
	}

	q := path.Query()
	q.Set("csrfToken", c.csrfToken)
	q.Set("locale", c.locale)
	q.Set("lang", c.lang)
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
	req = req.WithContext(ctx)

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

	var n int64
	var response StopInfo
	c.pool.Apply(func(buf *bytes.Buffer) {
		n, err = io.Copy(buf, reader)
		if err != nil {
			err = fmt.Errorf("copying data from body: %w", err)

			return
		}

		if n == 0 {
			return
		}

		c.logger.Debugf("response (%d): %s", buf.Len(), buf.String())

		if resp.StatusCode != http.StatusOK {
			c.logger.Debugf("status code is %d", resp.StatusCode)
			err = NewWrongStatusCodeError(resp.StatusCode)

			return
		}

		err = json.NewDecoder(buf).Decode(&response)
		if err != nil {
			err = fmt.Errorf("decoding response body: %w", err)

			return
		}
	})
	if err != nil {
		return StopInfo{}, false, err
	}

	if len(response.CsrfToken) != 0 {
		c.logger.Debugf("found new token, updating to %s", response.CsrfToken)
		c.csrfToken = response.CsrfToken
		c.logger.Debug("fetching info again")
		return StopInfo{}, true, nil
	}

	if response.Error != nil {
		return StopInfo{}, false, response.Error
	}

	c.logger.Debugf("fetched: %#+v", response)
	return response, false, nil
}
