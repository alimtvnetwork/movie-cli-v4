// schema_indexes.go — Index DDL split from schema.go for file size compliance.
package db

func (d *DB) createIndexes() error {
	_, err := d.Exec(`
	CREATE INDEX IF NOT EXISTS IdxCast_TmdbPersonId       ON Cast(TmdbPersonId);
	CREATE INDEX IF NOT EXISTS IdxCollection_TmdbCollectionId ON Collection(TmdbCollectionId);
	CREATE INDEX IF NOT EXISTS IdxScanHistory_ScanFolderId ON ScanHistory(ScanFolderId);
	CREATE INDEX IF NOT EXISTS IdxMedia_TmdbId             ON Media(TmdbId);
	CREATE INDEX IF NOT EXISTS IdxMedia_Type               ON Media(Type);
	CREATE INDEX IF NOT EXISTS IdxMedia_LanguageId         ON Media(LanguageId);
	CREATE INDEX IF NOT EXISTS IdxMedia_CollectionId       ON Media(CollectionId);
	CREATE INDEX IF NOT EXISTS IdxMedia_ScanHistoryId      ON Media(ScanHistoryId);
	CREATE INDEX IF NOT EXISTS IdxDirector_TmdbPersonId    ON Director(TmdbPersonId);
	CREATE INDEX IF NOT EXISTS IdxMediaDirector_MediaId    ON MediaDirector(MediaId);
	CREATE INDEX IF NOT EXISTS IdxMediaDirector_DirectorId ON MediaDirector(DirectorId);
	CREATE INDEX IF NOT EXISTS IdxMediaGenre_MediaId       ON MediaGenre(MediaId);
	CREATE INDEX IF NOT EXISTS IdxMediaGenre_GenreId        ON MediaGenre(GenreId);
	CREATE INDEX IF NOT EXISTS IdxMediaCast_MediaId        ON MediaCast(MediaId);
	CREATE INDEX IF NOT EXISTS IdxMediaCast_CastId         ON MediaCast(CastId);
	CREATE INDEX IF NOT EXISTS IdxMediaTag_MediaId         ON MediaTag(MediaId);
	CREATE INDEX IF NOT EXISTS IdxMediaTag_TagId           ON MediaTag(TagId);
	CREATE INDEX IF NOT EXISTS IdxMoveHistory_MediaId      ON MoveHistory(MediaId);
	CREATE INDEX IF NOT EXISTS IdxMoveHistory_FileActionId ON MoveHistory(FileActionId);
	CREATE INDEX IF NOT EXISTS IdxMoveHistory_IsReverted   ON MoveHistory(IsReverted);
	CREATE INDEX IF NOT EXISTS IdxActionHistory_FileActionId ON ActionHistory(FileActionId);
	CREATE INDEX IF NOT EXISTS IdxActionHistory_MediaId      ON ActionHistory(MediaId);
	CREATE INDEX IF NOT EXISTS IdxActionHistory_BatchId      ON ActionHistory(BatchId);
	CREATE INDEX IF NOT EXISTS IdxActionHistory_IsReverted   ON ActionHistory(IsReverted);
	CREATE INDEX IF NOT EXISTS IdxErrorLog_Level           ON ErrorLog(Level);
	CREATE INDEX IF NOT EXISTS IdxErrorLog_Command         ON ErrorLog(Command);
	CREATE INDEX IF NOT EXISTS IdxErrorLog_Timestamp       ON ErrorLog(Timestamp);
	CREATE INDEX IF NOT EXISTS IdxSeason_MediaId           ON Season(MediaId);
	CREATE INDEX IF NOT EXISTS IdxSeason_TmdbSeasonId      ON Season(TmdbSeasonId);
	CREATE INDEX IF NOT EXISTS IdxEpisode_SeasonId         ON Episode(SeasonId);
	CREATE INDEX IF NOT EXISTS IdxEpisode_TmdbEpisodeId    ON Episode(TmdbEpisodeId);
	CREATE INDEX IF NOT EXISTS IdxEpisode_IsWatched        ON Episode(IsWatched);
	`)
	return err
}
