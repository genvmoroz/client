package http

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

type (
	Client struct {
		client  *http.Client
		execute Execute
	}

	Execute func(*http.Client, *http.Request) (*http.Response, error)

	Option func(*Client) error
)

func NewClient(opts ...Option) (*Client, error) {
	client := &Client{
		client:  inanimateClient,
		execute: singleExecute(),
	}

	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, err
		}
	}

	return client, nil
}

func (c *Client) Get(url *url.URL) (*http.Response, error) {
	if url == nil {
		return nil, errors.New("url cannot be nil")
	}

	req := &http.Request{Method: http.MethodGet, URL: url, Header: make(map[string][]string)}
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Accept-Charset", "utf-8")

	return c.execute(c.client, req)
}

func WithRetry(retries uint, timeout time.Duration) Option {
	return func(client *Client) error {
		if timeout < 0 {
			return fmt.Errorf("negative timeout: %s", timeout)
		}

		client.execute = retryExecute(retries, timeout)
		return nil
	}
}

func WithClient(cli *http.Client) Option {
	return func(client *Client) error {
		if cli == nil {
			return errors.New("http client is required")
		}

		client.client = cli
		return nil
	}
}

var inanimateClient = &http.Client{
	Transport: &http.Transport{
		DisableKeepAlives: true,
	},
}

func singleExecute() Execute {
	return func(client *http.Client, request *http.Request) (*http.Response, error) {
		return client.Do(request)
	}
}

func retryExecute(retries uint, timeout time.Duration) Execute {
	return func(client *http.Client, request *http.Request) (*http.Response, error) {
		var (
			err  error
			resp *http.Response
		)
		for retry := int(retries); retry >= 0; retry-- {
			resp, err = client.Do(request)
			if err == nil {
				return resp, nil
			}
			if retry > 0 {
				logrus.Infof("failed to execute request: %s. Retrying", err.Error())
				time.Sleep(timeout)
			}
		}

		return nil, err
	}
}
