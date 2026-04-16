// types.go — TMDb API response types.
package tmdb

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
