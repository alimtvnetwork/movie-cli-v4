// Package tmdb provides a client for The Movie Database (TMDb) API.
package tmdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

const baseURL = "https://api.themoviedb.org/3"
const imageBaseURL = "https://image.tmdb.org/t/p/w500"

// Sentinel errors for callers to classify TMDb failures.
var (
	ErrAuthInvalid   = errors.New("TMDb API key is invalid")
	ErrAuthMissing   = errors.New("no TMDb API key configured")
	ErrRateLimited   = errors.New("TMDb rate limit exceeded")
	ErrServerError   = errors.New("TMDb is temporarily unavailable")
	ErrNetworkError  = errors.New("network error reaching TMDb")
	ErrTimeout       = errors.New("TMDb request timed out")
)

// Client interacts with the TMDb API.
type Client struct {
	HTTPClient  *http.Client
	APIKey      string
	AccessToken string
}

// NewClient creates a new TMDb client from an API key or env vars.
func NewClient(apiKey string) *Client {
	return NewClientWithToken(apiKey, "")
}

// NewClientWithToken creates a TMDb client using either an API key or bearer token.
func NewClientWithToken(apiKey, accessToken string) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("TMDB_API_KEY")
	}
	if accessToken == "" {
		accessToken = os.Getenv("TMDB_TOKEN")
	}
	return &Client{
		APIKey:      apiKey,
		AccessToken: accessToken,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// HasAuth returns true if the client has either an API key or access token.
func (c *Client) HasAuth() bool {
	return c.APIKey != "" || c.AccessToken != ""
}

// SearchMulti searches for movies and TV shows.
func (c *Client) SearchMulti(query string) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("page", "1")

	var resp searchResponse
	if err := c.get(c.buildURL("/search/multi", params), &resp); err != nil {
		return nil, err
	}

	var filtered []SearchResult
	for i := range resp.Results {
		if resp.Results[i].MediaType == "movie" || resp.Results[i].MediaType == "tv" {
			filtered = append(filtered, resp.Results[i])
		}
	}
	return filtered, nil
}

// GetMovieDetails returns detailed info for a movie.
func (c *Client) GetMovieDetails(tmdbID int) (*MovieDetails, error) {
	var d MovieDetails
	if err := c.get(c.buildURL(fmt.Sprintf("/movie/%d", tmdbID), nil), &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// GetTVDetails returns detailed info for a TV show.
func (c *Client) GetTVDetails(tmdbID int) (*TVDetails, error) {
	var d TVDetails
	if err := c.get(c.buildURL(fmt.Sprintf("/tv/%d", tmdbID), nil), &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// GetMovieCredits returns cast and crew for a movie.
func (c *Client) GetMovieCredits(tmdbID int) (*Credits, error) {
	var cr Credits
	if err := c.get(c.buildURL(fmt.Sprintf("/movie/%d/credits", tmdbID), nil), &cr); err != nil {
		return nil, err
	}
	return &cr, nil
}

// GetTVCredits returns cast and crew for a TV show.
func (c *Client) GetTVCredits(tmdbID int) (*Credits, error) {
	var cr Credits
	if err := c.get(c.buildURL(fmt.Sprintf("/tv/%d/credits", tmdbID), nil), &cr); err != nil {
		return nil, err
	}
	return &cr, nil
}

// GetMovieVideos returns videos (trailers, teasers) for a movie.
func (c *Client) GetMovieVideos(tmdbID int) ([]VideoResult, error) {
	var resp videosResponse
	if err := c.get(c.buildURL(fmt.Sprintf("/movie/%d/videos", tmdbID), nil), &resp); err != nil {
		return nil, err
	}
	return resp.Results, nil
}

// GetTVVideos returns videos (trailers, teasers) for a TV show.
func (c *Client) GetTVVideos(tmdbID int) ([]VideoResult, error) {
	var resp videosResponse
	if err := c.get(c.buildURL(fmt.Sprintf("/tv/%d/videos", tmdbID), nil), &resp); err != nil {
		return nil, err
	}
	return resp.Results, nil
}

// TrailerURL finds the best YouTube trailer URL from a list of videos.
func TrailerURL(videos []VideoResult) string {
	for _, v := range videos {
		if v.Site == "YouTube" && v.Type == "Trailer" {
			return "https://www.youtube.com/watch?v=" + v.Key
		}
	}
	for _, v := range videos {
		if v.Site == "YouTube" {
			return "https://www.youtube.com/watch?v=" + v.Key
		}
	}
	return ""
}

// DownloadPoster downloads a poster image and saves it to dst.
func (c *Client) DownloadPoster(posterPath, dst string) error {
	if posterPath == "" {
		return fmt.Errorf("no poster available")
	}

	imgURL := imageBaseURL + posterPath
	resp, err := c.HTTPClient.Get(imgURL)
	if err != nil {
		if IsNetworkError(err) {
			return fmt.Errorf("%w: %v", ErrNetworkError, err)
		}
		return err
	}
	defer resp.Body.Close()

	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// GetRecommendations returns recommended movies or TV shows.
func (c *Client) GetRecommendations(tmdbID int, mediaType string, page int) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("page", fmt.Sprintf("%d", page))

	var resp searchResponse
	if err := c.get(c.buildURL(fmt.Sprintf("/%s/%d/recommendations", mediaType, tmdbID), params), &resp); err != nil {
		return nil, err
	}
	return resp.Results, nil
}

// DiscoverByGenre discovers content by genre ID.
func (c *Client) DiscoverByGenre(mediaType string, genreID int, page int) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("with_genres", fmt.Sprintf("%d", genreID))
	params.Set("sort_by", "popularity.desc")
	params.Set("page", fmt.Sprintf("%d", page))

	var resp searchResponse
	if err := c.get(c.buildURL(fmt.Sprintf("/discover/%s", mediaType), params), &resp); err != nil {
		return nil, err
	}
	return resp.Results, nil
}

// Trending returns trending content.
func (c *Client) Trending(mediaType string) ([]SearchResult, error) {
	var resp searchResponse
	if err := c.get(c.buildURL(fmt.Sprintf("/trending/%s/week", mediaType), nil), &resp); err != nil {
		return nil, err
	}
	return resp.Results, nil
}

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
	return fmt.Errorf("TMDb request failed after %d retries: %w", MaxRetries, lastErr)
}

func (c *Client) doGet(reqURL string, target interface{}, attempt int) error {
	req, reqErr := http.NewRequest(http.MethodGet, reqURL, nil)
	if reqErr != nil {
		backoff(attempt)
		return fmt.Errorf("build request failed: %w", reqErr)
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
		return fmt.Errorf("%w: check your internet connection", ErrTimeout)
	}
	if IsNetworkError(err) {
		return fmt.Errorf("%w: %v", ErrNetworkError, err)
	}
	return fmt.Errorf("HTTP request failed: %w", err)
}

func handleResponse(resp *http.Response, target interface{}, attempt int) error {
	switch {
	case resp.StatusCode == 401:
		resp.Body.Close()
		return fmt.Errorf("%w. Run: movie config set tmdb_api_key YOUR_KEY", ErrAuthInvalid)

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
		lastErr := fmt.Errorf("%w (HTTP %d)", ErrServerError, resp.StatusCode)
		if attempt == 0 {
			delay := 3 * time.Second
			if resp.StatusCode == 502 || resp.StatusCode == 503 || resp.StatusCode == 504 {
				delay = 5 * time.Second
			}
			time.Sleep(delay)
			return lastErr
		}
		return lastErr

	case resp.StatusCode != 200:
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return fmt.Errorf("TMDb API error %d: %s", resp.StatusCode, string(body))
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
