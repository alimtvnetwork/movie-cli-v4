// Package tmdb provides a client for The Movie Database (TMDb) API.
package tmdb

import (
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

// SearchResult holds a search result from TMDb.
type SearchResult struct {
	Overview    string  `json:"overview"`
	Title       string  `json:"title"`
	Name        string  `json:"name"`
	ReleaseDate string  `json:"release_date"`
	FirstAir    string  `json:"first_air_date"`
	PosterPath  string  `json:"poster_path"`
	MediaType   string  `json:"media_type"`
	GenreIDs    []int   `json:"genre_ids"`
	VoteAvg     float64 `json:"vote_average"`
	Popularity  float64 `json:"popularity"`
	ID          int     `json:"id"`
}

type searchResponse struct {
	Results []SearchResult `json:"results"`
}

// MovieDetails holds detailed movie info.
type MovieDetails struct {
	Title            string  `json:"title"`
	Overview         string  `json:"overview"`
	ReleaseDate      string  `json:"release_date"`
	PosterPath       string  `json:"poster_path"`
	ImdbID           string  `json:"imdb_id"`
	OriginalLanguage string  `json:"original_language"`
	Tagline          string  `json:"tagline"`
	Genres           []Genre `json:"genres"`
	VoteAvg          float64 `json:"vote_average"`
	Popularity       float64 `json:"popularity"`
	ID               int     `json:"id"`
	Runtime          int     `json:"runtime"`
	Budget           int64   `json:"budget"`
	Revenue          int64   `json:"revenue"`
}

// VideoResult holds a single video from TMDb.
type VideoResult struct {
	Key  string `json:"key"`
	Site string `json:"site"`
	Type string `json:"type"`
	Name string `json:"name"`
}

type videosResponse struct {
	Results []VideoResult `json:"results"`
}

// TVDetails holds detailed TV show info.
type TVDetails struct {
	Name             string   `json:"name"`
	Overview         string   `json:"overview"`
	FirstAirDate     string   `json:"first_air_date"`
	PosterPath       string   `json:"poster_path"`
	OriginalLanguage string   `json:"original_language"`
	Tagline          string   `json:"tagline"`
	Genres           []Genre  `json:"genres"`
	Languages        []string `json:"languages"`
	VoteAvg          float64  `json:"vote_average"`
	Popularity       float64  `json:"popularity"`
	ID               int      `json:"id"`
	Seasons          int      `json:"number_of_seasons"`
	EpisodeRunTime   []int    `json:"episode_run_time"`
}

// Genre is a TMDb genre.
type Genre struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

// Credits holds cast and crew.
type Credits struct {
	Cast []CastMember `json:"cast"`
	Crew []CrewMember `json:"crew"`
}

// CastMember is a cast member.
type CastMember struct {
	Name      string `json:"name"`
	Character string `json:"character"`
	Order     int    `json:"order"`
}

// CrewMember is a crew member.
type CrewMember struct {
	Name string `json:"name"`
	Job  string `json:"job"`
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
