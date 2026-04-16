// season.go — Season and Episode DB helpers for TV show tracking.
package db

import "database/sql"

// Season represents a row in the Season table.
type Season struct {
	ID           int64
	MediaID      int64
	SeasonNumber int
	TmdbSeasonID int
	Name         string
	Overview     string
	PosterPath   string
	AirDate      string
	EpisodeCount int
}

// Episode represents a row in the Episode table.
type Episode struct {
	ID            int64
	SeasonID      int64
	EpisodeNumber int
	TmdbEpisodeID int
	Name          string
	Overview      string
	AirDate       string
	Runtime       int
	StillPath     string
	VoteAvg       float64
	IsWatched     bool
	WatchedAt     sql.NullString
}

// InsertSeason upserts a season for a media entry.
func (d *DB) InsertSeason(s *Season) (int64, error) {
	res, err := d.Exec(`
		INSERT INTO Season (MediaId, SeasonNumber, TmdbSeasonId, Name, Overview, PosterPath, AirDate, EpisodeCount)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (MediaId, SeasonNumber) DO UPDATE SET
			TmdbSeasonId=excluded.TmdbSeasonId, Name=excluded.Name,
			Overview=excluded.Overview, PosterPath=excluded.PosterPath,
			AirDate=excluded.AirDate, EpisodeCount=excluded.EpisodeCount`,
		s.MediaID, s.SeasonNumber, s.TmdbSeasonID, s.Name,
		s.Overview, s.PosterPath, s.AirDate, s.EpisodeCount)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	if id > 0 {
		return id, nil
	}
	// Get existing ID on conflict
	var seasonID int64
	err = d.QueryRow(
		"SELECT SeasonId FROM Season WHERE MediaId = ? AND SeasonNumber = ?",
		s.MediaID, s.SeasonNumber).Scan(&seasonID)
	return seasonID, err
}

// InsertEpisode upserts an episode for a season.
func (d *DB) InsertEpisode(e *Episode) (int64, error) {
	res, err := d.Exec(`
		INSERT INTO Episode (SeasonId, EpisodeNumber, TmdbEpisodeId, Name, Overview, AirDate, Runtime, StillPath, VoteAvg)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (SeasonId, EpisodeNumber) DO UPDATE SET
			TmdbEpisodeId=excluded.TmdbEpisodeId, Name=excluded.Name,
			Overview=excluded.Overview, AirDate=excluded.AirDate,
			Runtime=excluded.Runtime, StillPath=excluded.StillPath,
			VoteAvg=excluded.VoteAvg`,
		e.SeasonID, e.EpisodeNumber, e.TmdbEpisodeID, e.Name,
		e.Overview, e.AirDate, e.Runtime, e.StillPath, e.VoteAvg)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

// SeasonsByMediaID returns all seasons for a media entry.
func (d *DB) SeasonsByMediaID(mediaID int64) ([]Season, error) {
	rows, err := d.Query(`
		SELECT SeasonId, MediaId, SeasonNumber, COALESCE(TmdbSeasonId, 0),
		       COALESCE(Name, ''), COALESCE(Overview, ''), COALESCE(PosterPath, ''),
		       COALESCE(AirDate, ''), EpisodeCount
		FROM Season WHERE MediaId = ? ORDER BY SeasonNumber`, mediaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var seasons []Season
	for rows.Next() {
		var s Season
		if err := rows.Scan(&s.ID, &s.MediaID, &s.SeasonNumber, &s.TmdbSeasonID,
			&s.Name, &s.Overview, &s.PosterPath, &s.AirDate, &s.EpisodeCount); err != nil {
			return nil, err
		}
		seasons = append(seasons, s)
	}
	return seasons, nil
}

// EpisodesBySeasonID returns all episodes for a season.
func (d *DB) EpisodesBySeasonID(seasonID int64) ([]Episode, error) {
	rows, err := d.Query(`
		SELECT EpisodeId, SeasonId, EpisodeNumber, COALESCE(TmdbEpisodeId, 0),
		       COALESCE(Name, ''), COALESCE(Overview, ''), COALESCE(AirDate, ''),
		       Runtime, COALESCE(StillPath, ''), VoteAvg, IsWatched, WatchedAt
		FROM Episode WHERE SeasonId = ? ORDER BY EpisodeNumber`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []Episode
	for rows.Next() {
		var e Episode
		if err := rows.Scan(&e.ID, &e.SeasonID, &e.EpisodeNumber, &e.TmdbEpisodeID,
			&e.Name, &e.Overview, &e.AirDate, &e.Runtime, &e.StillPath,
			&e.VoteAvg, &e.IsWatched, &e.WatchedAt); err != nil {
			return nil, err
		}
		episodes = append(episodes, e)
	}
	return episodes, nil
}

// MarkEpisodeWatched marks an episode as watched.
func (d *DB) MarkEpisodeWatched(episodeID int64) error {
	_, err := d.Exec(
		"UPDATE Episode SET IsWatched = 1, WatchedAt = datetime('now') WHERE EpisodeId = ?",
		episodeID)
	return err
}

// MarkEpisodeUnwatched reverts an episode to unwatched.
func (d *DB) MarkEpisodeUnwatched(episodeID int64) error {
	_, err := d.Exec(
		"UPDATE Episode SET IsWatched = 0, WatchedAt = NULL WHERE EpisodeId = ?",
		episodeID)
	return err
}
