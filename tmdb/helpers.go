// helpers.go — utility functions, genre maps, and error classification.
package tmdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// GetDisplayTitle returns the correct display title (title for movies, name for TV).
func (r *SearchResult) GetDisplayTitle() string {
	if r.Title != "" {
		return r.Title
	}
	return r.Name
}

// GetYear extracts year from release_date or first_air_date.
func (r *SearchResult) GetYear() string {
	date := r.ReleaseDate
	if date == "" {
		date = r.FirstAir
	}
	if len(date) >= 4 {
		return date[:4]
	}
	return ""
}

// GenreNames converts genre IDs to names.
func GenreNames(ids []int) string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if n, ok := genreMap[id]; ok {
			names = append(names, n)
		}
	}
	return strings.Join(names, ", ")
}

// PosterURL returns the full poster URL.
func PosterURL(path string) string {
	if path == "" {
		return ""
	}
	return imageBaseURL + path
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

// MaxRetries is the number of retry attempts for rate-limited requests.
const MaxRetries = 3

// IsNetworkError returns true if the error is a network-level failure (DNS, connection, timeout).
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	// Check for common network error strings as fallback
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "network is unreachable")
}

// IsTimeoutError returns true if the error is specifically a timeout.
func IsTimeoutError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return false
}

func (c *Client) get(reqURL string, target interface{}) error {
	var lastErr error
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		req, reqErr := http.NewRequest(http.MethodGet, reqURL, nil)
		if reqErr != nil {
			lastErr = fmt.Errorf("build request failed: %w", reqErr)
			backoff(attempt)
			continue
		}
		if c.AccessToken != "" {
			req.Header.Set("Authorization", "Bearer "+c.AccessToken)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			// Classify network errors per spec §1.4 and §4
			if IsTimeoutError(err) {
				return fmt.Errorf("%w: check your internet connection", ErrTimeout)
			}
			if IsNetworkError(err) {
				return fmt.Errorf("%w: %v", ErrNetworkError, err)
			}
			lastErr = fmt.Errorf("HTTP request failed: %w", err)
			backoff(attempt)
			continue
		}

		// Handle specific HTTP status codes per spec
		switch {
		case resp.StatusCode == 401:
			resp.Body.Close()
			return fmt.Errorf("%w. Run: movie config set tmdb_api_key YOUR_KEY", ErrAuthInvalid)

		case resp.StatusCode == 429:
			// Rate limit — retry per spec §1.1
			resp.Body.Close()
			lastErr = ErrRateLimited
			retryAfter := resp.Header.Get("Retry-After")
			delay := 2 * time.Second
			if secs, parseErr := time.ParseDuration(retryAfter + "s"); parseErr == nil && secs > 0 {
				delay = secs
			}
			time.Sleep(delay)
			continue

		case resp.StatusCode >= 500:
			// Server errors — retry once per spec §1.3
			resp.Body.Close()
			lastErr = fmt.Errorf("%w (HTTP %d)", ErrServerError, resp.StatusCode)
			if attempt == 0 {
				delay := 3 * time.Second
				if resp.StatusCode == 502 || resp.StatusCode == 503 || resp.StatusCode == 504 {
					delay = 5 * time.Second
				}
				time.Sleep(delay)
				continue
			}
			return lastErr

		case resp.StatusCode != 200:
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("TMDb API error %d: %s", resp.StatusCode, string(body))
		}

		err = json.NewDecoder(resp.Body).Decode(target)
		resp.Body.Close()
		return err
	}
	return fmt.Errorf("TMDb request failed after %d retries: %w", MaxRetries, lastErr)
}

// backoff sleeps for exponential duration: 1s, 2s, 4s, ...
func backoff(attempt int) {
	if attempt >= MaxRetries {
		return
	}
	d := time.Duration(1<<uint(attempt)) * time.Second
	time.Sleep(d)
}

// genreMap maps TMDb genre IDs to names (combined movie + TV).
var genreMap = map[int]string{
	28: "Action", 12: "Adventure", 16: "Animation", 35: "Comedy",
	80: "Crime", 99: "Documentary", 18: "Drama", 10751: "Family",
	14: "Fantasy", 36: "History", 27: "Horror", 10402: "Music",
	9648: "Mystery", 10749: "Romance", 878: "Science Fiction",
	10770: "TV Movie", 53: "Thriller", 10752: "War", 37: "Western",
	10759: "Action & Adventure", 10762: "Kids", 10763: "News",
	10764: "Reality", 10765: "Sci-Fi & Fantasy", 10766: "Soap",
	10767: "Talk", 10768: "War & Politics",
}

// GenreNameToID returns a reverse map of genre name → TMDb genre ID.
func GenreNameToID() map[string]int {
	m := make(map[string]int, len(genreMap))
	for id, name := range genreMap {
		m[name] = id
	}
	return m
}
