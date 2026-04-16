// constants.go — shared constants for media types, output formats, and watch statuses.
package db

// MediaType represents the type of media content.
type MediaType string

const (
	MediaTypeMovie MediaType = "movie"
	MediaTypeTV    MediaType = "tv"
)

// String returns the string value of MediaType.
func (mt MediaType) String() string {
	return string(mt)
}

// OutputFormat represents the CLI output display format.
type OutputFormat string

const (
	OutputFormatDefault OutputFormat = ""
	OutputFormatJSON    OutputFormat = "json"
	OutputFormatTable   OutputFormat = "table"
)

// String returns the string value of OutputFormat.
func (of OutputFormat) String() string {
	return string(of)
}

// WatchStatus represents the status of a watchlist entry.
type WatchStatus string

const (
	WatchStatusToWatch WatchStatus = "to-watch"
	WatchStatusWatched WatchStatus = "watched"
)

// String returns the string value of WatchStatus.
func (ws WatchStatus) String() string {
	return string(ws)
}

// JSONSubDir returns the JSON output subdirectory name for a media type.
func JSONSubDir(mediaType string) string {
	if mediaType == string(MediaTypeTV) {
		return string(MediaTypeTV)
	}
	return string(MediaTypeMovie)
}

// TypeIcon returns the emoji icon for a media type.
func TypeIcon(mediaType string) string {
	if mediaType == string(MediaTypeTV) {
		return "📺"
	}
	return "🎬"
}

// TypeLabel returns the display label for a media type.
func TypeLabel(mediaType string) string {
	if mediaType == string(MediaTypeTV) {
		return "TV Show"
	}
	return "Movie"
}

// TypeLabelPlural returns the plural display label for a media type.
func TypeLabelPlural(mediaType string) string {
	if mediaType == string(MediaTypeTV) {
		return "TV Shows"
	}
	return "Movies"
}
