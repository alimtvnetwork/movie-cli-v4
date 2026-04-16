// helpers.go — genre maps, poster URL, network error classification.
package tmdb

import (
	"errors"
	"net"
	"strings"
)

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

// IsNetworkError returns true if the error is a network-level failure.
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

// GenreNameToID returns a reverse map of genre name → TMDb genre ID.
func GenreNameToID() map[string]int {
	m := make(map[string]int, len(genreMap))
	for id, name := range genreMap {
		m[name] = id
	}
	return m
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
