// views.go — Database view creation.
package db

// createViews creates or replaces all database views.
func (d *DB) createViews() error {
	_, err := d.Exec(`
	CREATE VIEW IF NOT EXISTS VwMediaDetail AS
	SELECT
		m.MediaId, m.Title, m.CleanTitle, m.Year, m.Type,
		m.TmdbId, m.ImdbId, m.Description,
		m.ImdbRating, m.TmdbRating, m.Popularity,
		l.Code AS LanguageCode, l.Name AS LanguageName,
		m.Director, m.ThumbnailPath,
		m.OriginalFileName, m.OriginalFilePath, m.CurrentFilePath,
		m.FileExtension, m.FileSizeMb, m.Runtime,
		m.Budget, m.Revenue, m.TrailerUrl, m.Tagline,
		m.ScanHistoryId, m.ScannedAt, m.UpdatedAt
	FROM Media m
	LEFT JOIN Language l ON m.LanguageId = l.LanguageId;

	CREATE VIEW IF NOT EXISTS VwMediaGenreList AS
	SELECT
		mg.MediaGenreId, mg.MediaId,
		g.GenreId, g.Name AS GenreName
	FROM MediaGenre mg
	INNER JOIN Genre g ON mg.GenreId = g.GenreId;

	CREATE VIEW IF NOT EXISTS VwMediaCastList AS
	SELECT
		mc.MediaCastId, mc.MediaId,
		c.CastId, c.Name AS CastName, c.TmdbPersonId,
		mc.Role, mc.CastOrder
	FROM MediaCast mc
	INNER JOIN Cast c ON mc.CastId = c.CastId;

	CREATE VIEW IF NOT EXISTS VwMediaFull AS
	SELECT
		m.MediaId, m.Title, m.CleanTitle, m.Year, m.Type,
		m.TmdbId, m.ImdbId, m.Description,
		m.ImdbRating, m.TmdbRating, m.Popularity,
		l.Code AS LanguageCode, l.Name AS LanguageName,
		m.Director, m.ThumbnailPath, m.CurrentFilePath,
		m.FileExtension, m.FileSizeMb, m.Runtime,
		m.TrailerUrl, m.Tagline, m.ScannedAt, m.UpdatedAt,
		COALESCE(
			(SELECT GROUP_CONCAT(g.Name, ', ')
			 FROM MediaGenre mg
			 INNER JOIN Genre g ON mg.GenreId = g.GenreId
			 WHERE mg.MediaId = m.MediaId), ''
		) AS Genres,
		COALESCE(
			(SELECT GROUP_CONCAT(c.Name, ', ')
			 FROM MediaCast mc
			 INNER JOIN Cast c ON mc.CastId = c.CastId
			 WHERE mc.MediaId = m.MediaId
			 ORDER BY mc.CastOrder), ''
		) AS CastList
	FROM Media m
	LEFT JOIN Language l ON m.LanguageId = l.LanguageId;

	CREATE VIEW IF NOT EXISTS VwMoveHistoryDetail AS
	SELECT
		mh.MoveHistoryId, mh.MediaId,
		m.Title AS MediaTitle,
		fa.Name AS ActionName,
		mh.FromPath, mh.ToPath,
		mh.OriginalFileName, mh.NewFileName,
		mh.IsReverted, mh.MovedAt
	FROM MoveHistory mh
	INNER JOIN Media m ON mh.MediaId = m.MediaId
	INNER JOIN FileAction fa ON mh.FileActionId = fa.FileActionId;

	CREATE VIEW IF NOT EXISTS VwActionHistoryDetail AS
	SELECT
		ah.ActionHistoryId, ah.MediaId,
		m.Title AS MediaTitle,
		fa.Name AS ActionName,
		ah.MediaSnapshot, ah.Detail, ah.BatchId,
		ah.IsReverted, ah.CreatedAt
	FROM ActionHistory ah
	INNER JOIN FileAction fa ON ah.FileActionId = fa.FileActionId
	LEFT JOIN Media m ON ah.MediaId = m.MediaId;

	CREATE VIEW IF NOT EXISTS VwScanHistoryDetail AS
	SELECT
		sh.ScanHistoryId, sh.ScanFolderId,
		sf.FolderPath, sf.IsActive AS FolderIsActive,
		sh.TotalFiles, sh.MoviesFound, sh.TvFound,
		sh.NewFiles, sh.RemovedFiles, sh.UpdatedFiles,
		sh.ErrorCount, sh.DurationMs, sh.ScannedAt
	FROM ScanHistory sh
	INNER JOIN ScanFolder sf ON sh.ScanFolderId = sf.ScanFolderId;

	CREATE VIEW IF NOT EXISTS VwMediaTag AS
	SELECT
		mt.MediaTagId, mt.MediaId,
		m.Title AS MediaTitle,
		t.TagId, t.Name AS TagName,
		mt.CreatedAt
	FROM MediaTag mt
	INNER JOIN Media m ON mt.MediaId = m.MediaId
	INNER JOIN Tag t   ON mt.TagId   = t.TagId;
	`)
	return err
}
