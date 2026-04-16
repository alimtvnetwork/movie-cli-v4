package tmdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/alimtvnetwork/movie-cli-v4/apperror"
)

func (c *Client) buildURL(path string, params url.Values) string {
	if params == nil {
		params = url.Values{}
	}
	if c.AccessToken == "" && c.APIKey != "" {
		params.Set("api_key", c.APIKey)
	}
	encoded := params.Encode()
	if encoded == "" {
		return baseURL + path
	}
	return baseURL + path + "?" + encoded
}

// MaxRetries is the number of retry attempts for rate-limited requests.
const MaxRetries = 3

func (c *Client) get(reqURL string, target interface{}) error {
	var lastErr error
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		lastErr = c.doGet(reqURL, target, attempt)
		if lastErr == nil {
			return nil
		}
		if lastErr == ErrRateLimited {
			continue
		}
		if errors.Is(lastErr, ErrTimeout) || errors.Is(lastErr, ErrNetworkError) || errors.Is(lastErr, ErrAuthInvalid) {
			return lastErr
		}
	}
	return apperror.Wrapf(lastErr, "TMDb request failed after %d retries", MaxRetries)
}

func (c *Client) doGet(reqURL string, target interface{}, attempt int) error {
	req, reqErr := http.NewRequest(http.MethodGet, reqURL, nil)
	if reqErr != nil {
		backoff(attempt)
		return apperror.Wrap("build request failed", reqErr)
	}
	if c.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return classifyHTTPError(err)
	}

	return handleResponse(resp, target, attempt)
}

func classifyHTTPError(err error) error {
	if IsTimeoutError(err) {
		return apperror.New("%w: check your internet connection", ErrTimeout)
	}
	if IsNetworkError(err) {
		return apperror.Wrapf(err, "%w", ErrNetworkError)
	}
	return apperror.Wrap("HTTP request failed", err)
}

func handleResponse(resp *http.Response, target interface{}, attempt int) error {
	switch {
	case resp.StatusCode == 401:
		resp.Body.Close()
		return apperror.New("%w. Run: movie config set tmdb_api_key YOUR_KEY", ErrAuthInvalid)

	case resp.StatusCode == 429:
		resp.Body.Close()
		retryAfter := resp.Header.Get("Retry-After")
		delay := 2 * time.Second
		if secs, parseErr := time.ParseDuration(retryAfter + "s"); parseErr == nil && secs > 0 {
			delay = secs
		}
		time.Sleep(delay)
		return ErrRateLimited

	case resp.StatusCode >= 500:
		resp.Body.Close()
		lastErr := apperror.New("%w (HTTP %d)", ErrServerError, resp.StatusCode)
		if attempt == 0 {
			delay := serverRetryDelay(resp.StatusCode)
			time.Sleep(delay)
		}
		return lastErr

	case resp.StatusCode != 200:
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return apperror.New("TMDb API error %d: %s", resp.StatusCode, string(body))
	}

	err := json.NewDecoder(resp.Body).Decode(target)
	resp.Body.Close()
	return err
}

// backoff sleeps for exponential duration: 1s, 2s, 4s, ...
func backoff(attempt int) {
	if attempt >= MaxRetries {
		return
	}
	d := time.Duration(1<<uint(attempt)) * time.Second
	time.Sleep(d)
}
