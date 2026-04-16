package db

// Media represents a row in the Media table.
type Media struct {
	Title            string
	CleanTitle       string
	Type             string // MediaTypeMovie or MediaTypeTV
	ImdbID           string
	Description      string
	Director         string
	ThumbnailPath    string
	OriginalFileName string
	OriginalFilePath string
	CurrentFilePath  string
	FileExtension    string
	TrailerURL       string
	Tagline          string
	ID               int64
	Budget           int64
	Revenue          int64
	ImdbRating       float64
	TmdbRating       float64
	Popularity       float64
	FileSizeMb       float64
	Year             int
	TmdbID           int
	Runtime          int
	LanguageId       int
	CollectionId     int
	ScanHistoryId    int

	// Compat fields — populated from views or for legacy cmd code.
	// These are NOT stored in the Media table directly.
	Genre    string `json:"genre,omitempty"`    // aggregated from MediaGenre+Genre
	CastList string `json:"cast_list,omitempty"` // aggregated from MediaCast+Cast
	Language string `json:"language,omitempty"` // resolved from Language.Code
	FileSize int64  `json:"file_size,omitempty"` // computed: FileSizeMb * 1024 * 1024
}

// mediaColumns is the standard SELECT column list for Media queries.
const mediaColumns = `MediaId, Title, CleanTitle, Year, Type,
	COALESCE(TmdbId, 0), COALESCE(ImdbId, ''), COALESCE(Description, ''),
	COALESCE(ImdbRating, 0), COALESCE(TmdbRating, 0), COALESCE(Popularity, 0),
	COALESCE(LanguageId, 0), COALESCE(CollectionId, 0),
	COALESCE(Director, ''), COALESCE(ThumbnailPath, ''),
	COALESCE(OriginalFileName, ''), COALESCE(OriginalFilePath, ''),
	COALESCE(CurrentFilePath, ''), COALESCE(FileExtension, ''),
	COALESCE(FileSizeMb, 0),
	COALESCE(Runtime, 0), COALESCE(Budget, 0), COALESCE(Revenue, 0),
	COALESCE(TrailerUrl, ''), COALESCE(Tagline, ''),
	COALESCE(ScanHistoryId, 0)`

// InsertMedia inserts a new media record and returns the ID.
func (d *DB) InsertMedia(m *Media) (int64, error) {
	var tmdbID interface{}
	if m.TmdbID > 0 {
		tmdbID = m.TmdbID
	}
	var langID interface{}
	if m.LanguageId > 0 {
		langID = m.LanguageId
	}
	var collID interface{}
	if m.CollectionId > 0 {
		collID = m.CollectionId
	}
	var scanID interface{}
	if m.ScanHistoryId > 0 {
		scanID = m.ScanHistoryId
	}

	res, err := d.Exec(`
		INSERT INTO Media (Title, CleanTitle, Year, Type, TmdbId, ImdbId,
			Description, ImdbRating, TmdbRating, Popularity, LanguageId, CollectionId,
			Director, ThumbnailPath, OriginalFileName, OriginalFilePath,
			CurrentFilePath, FileExtension, FileSizeMb,
			Runtime, Budget, Revenue, TrailerUrl, Tagline, ScanHistoryId)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.Title, m.CleanTitle, m.Year, m.Type, tmdbID, m.ImdbID,
		m.Description, m.ImdbRating, m.TmdbRating, m.Popularity, langID, collID,
		m.Director, m.ThumbnailPath, m.OriginalFileName, m.OriginalFilePath,
		m.CurrentFilePath, m.FileExtension, m.FileSizeMb,
		m.Runtime, m.Budget, m.Revenue, m.TrailerURL, m.Tagline, scanID,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateMediaByID updates an existing record by its primary key.
func (d *DB) UpdateMediaByID(m *Media) error {
	var tmdbID interface{}
	if m.TmdbID > 0 {
		tmdbID = m.TmdbID
	}
	var langID interface{}
	if m.LanguageId > 0 {
		langID = m.LanguageId
	}
	var collID interface{}
	if m.CollectionId > 0 {
		collID = m.CollectionId
	}
	_, err := d.Exec(`
		UPDATE Media SET Title=?, CleanTitle=?, Year=?, Type=?, TmdbId=?, ImdbId=?,
			Description=?, ImdbRating=?, TmdbRating=?, Popularity=?, LanguageId=?, CollectionId=?,
			Director=?, ThumbnailPath=?, CurrentFilePath=?,
			FileExtension=?, FileSizeMb=?,
			Runtime=?, Budget=?, Revenue=?, TrailerUrl=?, Tagline=?,
			UpdatedAt=datetime('now')
		WHERE MediaId=?`,
		m.Title, m.CleanTitle, m.Year, m.Type, tmdbID, m.ImdbID,
		m.Description, m.ImdbRating, m.TmdbRating, m.Popularity, langID, collID,
		m.Director, m.ThumbnailPath, m.CurrentFilePath,
		m.FileExtension, m.FileSizeMb,
		m.Runtime, m.Budget, m.Revenue, m.TrailerURL, m.Tagline,
		m.ID,
	)
	return err
}

// UpdateMediaByTmdbID updates an existing record matched by TmdbId.
func (d *DB) UpdateMediaByTmdbID(m *Media) error {
	var langID interface{}
	if m.LanguageId > 0 {
		langID = m.LanguageId
	}
	var collID interface{}
	if m.CollectionId > 0 {
		collID = m.CollectionId
	}
	_, err := d.Exec(`
		UPDATE Media SET Title=?, CleanTitle=?, Year=?, Type=?, ImdbId=?,
			Description=?, ImdbRating=?, TmdbRating=?, Popularity=?, LanguageId=?, CollectionId=?,
			Director=?, ThumbnailPath=?, CurrentFilePath=?,
			FileExtension=?, FileSizeMb=?,
			Runtime=?, Budget=?, Revenue=?, TrailerUrl=?, Tagline=?,
			UpdatedAt=datetime('now')
		WHERE TmdbId=?`,
		m.Title, m.CleanTitle, m.Year, m.Type, m.ImdbID,
		m.Description, m.ImdbRating, m.TmdbRating, m.Popularity, langID, collID,
		m.Director, m.ThumbnailPath, m.CurrentFilePath,
		m.FileExtension, m.FileSizeMb,
		m.Runtime, m.Budget, m.Revenue, m.TrailerURL, m.Tagline,
		m.TmdbID,
	)
	return err
}

// UpdateMediaPath updates the current file path.
func (d *DB) UpdateMediaPath(mediaID int64, newPath string) error {
	_, err := d.Exec("UPDATE Media SET CurrentFilePath = ?, UpdatedAt = datetime('now') WHERE MediaId = ?", newPath, mediaID)
	return err
}
