// movie_tmdb.go — TMDb credential helpers for interactive commands
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
)

type tmdbCredentials struct {
	APIKey string
	Token  string
}

func (c tmdbCredentials) HasAuth() bool {
	return c.APIKey != "" || c.Token != ""
}

// resolveScanTMDbCredentials loads saved/env credentials or prompts before scan.
func resolveScanTMDbCredentials(database *db.DB) tmdbCredentials {
	creds := readTMDbCredentials(database)
	if creds.HasAuth() {
		return creds
	}

	fmt.Println("⚠️  TMDb is not configured yet.")
	fmt.Println("   Enter your TMDb API key and/or TMDb access token before scanning.")
	fmt.Println("   Leave both blank to continue without metadata.")

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("   TMDb API key: ")
	if scanner.Scan() {
		creds.APIKey = strings.TrimSpace(scanner.Text())
	}

	fmt.Print("   TMDb access token: ")
	if scanner.Scan() {
		creds.Token = strings.TrimSpace(scanner.Text())
	}
	fmt.Println()

	if creds.APIKey != "" {
		if err := database.SetConfig("TmdbApiKey", creds.APIKey); err != nil {
			errlog.Warn("Could not save tmdb_api_key: %v", err)
		}
	}
	if creds.Token != "" {
		if err := database.SetConfig("TmdbToken", creds.Token); err != nil {
			errlog.Warn("Could not save tmdb_token: %v", err)
		}
	}

	if creds.HasAuth() {
		fmt.Println("✅ TMDb credentials saved.")
		fmt.Println()
		return creds
	}
	fmt.Println("⚠️  No TMDb credentials provided.")
	fmt.Println("   Scanning will continue without metadata fetching.")
	fmt.Println()

	return creds
}

// readTMDbCredentials reads TMDb credentials from config first, then env.
func readTMDbCredentials(database *db.DB) tmdbCredentials {
	creds := tmdbCredentials{
		APIKey: strings.TrimSpace(readTMDbConfigValue(database, "TmdbApiKey")),
		Token:  strings.TrimSpace(readTMDbConfigValue(database, "TmdbToken")),
	}
	if creds.APIKey == "" {
		creds.APIKey = strings.TrimSpace(os.Getenv("TMDB_API_KEY"))
	}
	if creds.Token == "" {
		creds.Token = strings.TrimSpace(os.Getenv("TMDB_TOKEN"))
	}
	return creds
}

func readTMDbConfigValue(database *db.DB, key string) string {
	val, err := database.GetConfig(key)
	if err != nil {
		if err.Error() != "sql: no rows in result set" {
			errlog.Warn("Config read error for %s: %v", key, err)
		}
		return ""
	}
	return val
}
