// movie_export.go — movie export
// Dumps the media table as JSON with optional storage stats and genre breakdown.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

var exportOutput string

var movieExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export media library as JSON",
	Long: `Dump the entire media table to a JSON file with storage stats and genre breakdown.

Default output: ./data/json/export/media.json

Examples:
  movie export                              # Export to default path
  movie export -o ~/Desktop/library.json    # Custom output path`,
	Run: runExport,
}

func init() {
	movieExportCmd.Flags().StringVarP(&exportOutput, "output", "o", "",
		"Output file path (default: ./data/json/export/media.json)")
}

// exportEnvelope is the top-level JSON structure.
type exportEnvelope struct {
	Meta    exportMeta        `json:"meta"`
	Storage *exportStorage    `json:"storage,omitempty"`
	Genres  []exportGenre     `json:"genres,omitempty"`
	Media   []exportMediaJSON `json:"media"`
}

type exportMeta struct {
	TotalItems  int    `json:"total_items"`
	TotalMovies int    `json:"total_movies"`
	TotalTV     int    `json:"total_tv_shows"`
	ExportedAt  string `json:"exported_at"`
}

type exportStorage struct {
	TotalSizeMb  float64 `json:"total_size_mb"`
	TotalHuman   string  `json:"total_human"`
	LargestMb    float64 `json:"largest_file_mb"`
	LargestTitle string  `json:"largest_file_title,omitempty"`
	SmallestMb   float64 `json:"smallest_file_mb"`
	AverageMb    float64 `json:"average_file_mb"`
}

type exportGenre struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// exportMediaJSON mirrors db.Media with JSON tags for clean output.
type exportMediaJSON struct {
	Title            string  `json:"title"`
	CleanTitle       string  `json:"clean_title"`
	Type             string  `json:"type"`
	ImdbID           string  `json:"imdb_id,omitempty"`
	Description      string  `json:"description,omitempty"`
	Genre            string  `json:"genre,omitempty"`
	Director         string  `json:"director,omitempty"`
	CastList         string  `json:"cast_list,omitempty"`
	ThumbnailPath    string  `json:"thumbnail_path,omitempty"`
	OriginalFileName string  `json:"original_file_name,omitempty"`
	OriginalFilePath string  `json:"original_file_path,omitempty"`
	CurrentFilePath  string  `json:"current_file_path,omitempty"`
	FileExtension    string  `json:"file_extension,omitempty"`
	Language         string  `json:"language,omitempty"`
	TrailerURL       string  `json:"trailer_url,omitempty"`
	Tagline          string  `json:"tagline,omitempty"`
	ID               int64   `json:"id"`
	FileSizeMb       float64 `json:"file_size_mb,omitempty"`
	Budget           int64   `json:"budget,omitempty"`
	Revenue          int64   `json:"revenue,omitempty"`
	ImdbRating       float64 `json:"imdb_rating,omitempty"`
	TmdbRating       float64 `json:"tmdb_rating,omitempty"`
	Popularity       float64 `json:"popularity,omitempty"`
	Year             int     `json:"year"`
	TmdbID           int     `json:"tmdb_id"`
	Runtime          int     `json:"runtime,omitempty"`
}

func toExportMediaJSON(m db.Media) exportMediaJSON {
	return exportMediaJSON{
		ID: m.ID, Title: m.Title, CleanTitle: m.CleanTitle,
		Year: m.Year, Type: m.Type, TmdbID: m.TmdbID, ImdbID: m.ImdbID,
		Description: m.Description, ImdbRating: m.ImdbRating, TmdbRating: m.TmdbRating,
		Popularity: m.Popularity, Genre: m.Genre, Director: m.Director,
		CastList: m.CastList, ThumbnailPath: m.ThumbnailPath,
		OriginalFileName: m.OriginalFileName, OriginalFilePath: m.OriginalFilePath,
		CurrentFilePath: m.CurrentFilePath, FileExtension: m.FileExtension,
		FileSizeMb: m.FileSizeMb, Runtime: m.Runtime, Language: m.Language,
		Budget: m.Budget, Revenue: m.Revenue, TrailerURL: m.TrailerURL,
		Tagline: m.Tagline,
	}
}

func runExport(cmd *cobra.Command, args []string) {
	database, err := db.Open()
	if err != nil {
		errlog.Error("Database error: %v", err)
		return
	}
	defer database.Close()

	items, err := database.ListMedia(0, 100000)
	if err != nil {
		errlog.Error("Failed to read media: %v", err)
		return
	}
	if len(items) == 0 {
		fmt.Println("📭 No media to export. Run 'movie scan <folder>' first.")
		return
	}

	envelope := buildExportEnvelope(database, items)
	writeExportFile(envelope, len(items))
}

func buildExportEnvelope(database *db.DB, items []db.Media) exportEnvelope {
	mediaOut := make([]exportMediaJSON, len(items))
	for i := range items {
		mediaOut[i] = toExportMediaJSON(items[i])
	}

	totalMovies, _ := database.CountMedia(string(db.MediaTypeMovie))
	totalTV, _ := database.CountMedia(string(db.MediaTypeTV))

	envelope := exportEnvelope{
		Meta: exportMeta{
			TotalItems: len(items), TotalMovies: totalMovies,
			TotalTV: totalTV, ExportedAt: db.NowUTC(),
		},
		Media: mediaOut,
	}

	envelope.Storage = buildExportStorage(database, items)
	envelope.Genres = buildExportGenres(database)
	return envelope
}

func buildExportStorage(database *db.DB, items []db.Media) *exportStorage {
	totalSize, largest, smallest, sizeErr := database.FileSizeStats()
	if sizeErr != nil || totalSize <= 0 {
		return nil
	}
	st := &exportStorage{
		TotalSizeMb: totalSize, TotalHuman: db.HumanSize(totalSize),
		LargestMb: largest, SmallestMb: smallest,
	}
	if len(items) > 0 {
		st.AverageMb = totalSize / float64(len(items))
	}
	for _, m := range items {
		if m.FileSizeMb == largest {
			st.LargestTitle = m.CleanTitle
			break
		}
	}
	return st
}

func buildExportGenres(database *db.DB) []exportGenre {
	genres, genreErr := database.TopGenres(50)
	if genreErr != nil || len(genres) == 0 {
		return nil
	}
	var out []exportGenre
	for name, count := range genres {
		out = append(out, exportGenre{Name: name, Count: count})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	return out
}

func writeExportFile(envelope exportEnvelope, itemCount int) {
	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		errlog.Error("JSON encoding error: %v", err)
		return
	}

	outPath := exportOutput
	if outPath == "" {
		outPath = filepath.Join(".", "data", "json", "export", "media.json")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		errlog.Error("Cannot create directory: %v", err)
		return
	}
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		errlog.Error("Failed to write file: %v", err)
		return
	}
	fmt.Printf("✅ Exported %d items → %s\n", itemCount, outPath)
}
