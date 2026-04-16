// movie_tag.go — movie tag [add|remove|list]
package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/alimtvnetwork/movie-cli-v4/db"
	"github.com/alimtvnetwork/movie-cli-v4/errlog"
	"github.com/spf13/cobra"
)

// movie tag — parent command
var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage tags on media items",
	Long:  "Add, remove, or list user-defined tags on movies and TV shows.",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

// movie tag add <id> <tag>
var tagAddCmd = &cobra.Command{
	Use:   "add <id> <tag>",
	Short: "Add a tag to a media item",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		d, err := db.Open()
		if err != nil {
			errlog.Error("Database error: %v", err)
			return
		}
		defer d.Close()

		id, err := strconv.Atoi(args[0])
		if err != nil {
			errlog.Error("Invalid ID: %s", args[0])
			return
		}
		tag := strings.TrimSpace(args[1])
		if tag == "" {
			errlog.Error("Tag cannot be empty")
			return
		}

		media, err := d.GetMediaByID(int64(id))
		if err != nil || media == nil {
			errlog.Error("Media not found with ID %d", id)
			return
		}

		err = d.AddTag(id, tag)
		if err != nil && strings.Contains(err.Error(), "UNIQUE constraint") {
			errlog.Error("Tag \"%s\" already exists on \"%s\"", tag, media.Title)
			return
		}
		if err != nil {
			errlog.Error("Error adding tag: %v", err)
			return
		}

		year := ""
		if media.Year > 0 {
			year = fmt.Sprintf(" (%d)", media.Year)
		}
		fmt.Printf("✅ Tag \"%s\" added to \"%s%s\"\n", tag, media.Title, year)
	},
}

// movie tag remove <id> <tag>
var tagRemoveCmd = &cobra.Command{
	Use:   "remove <id> <tag>",
	Short: "Remove a tag from a media item",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		d, err := db.Open()
		if err != nil {
			errlog.Error("Database error: %v", err)
			return
		}
		defer d.Close()

		id, err := strconv.Atoi(args[0])
		if err != nil {
			errlog.Error("Invalid ID: %s", args[0])
			return
		}
		tag := strings.TrimSpace(args[1])

		media, err := d.GetMediaByID(int64(id))
		if err != nil || media == nil {
			errlog.Error("Media not found with ID %d", id)
			return
		}

		removed, err := d.RemoveTag(id, tag)
		if err != nil {
			errlog.Error("Error removing tag: %v", err)
			return
		}
		if !removed {
			errlog.Error("Tag \"%s\" not found on \"%s\"", tag, media.Title)
			return
		}

		year := ""
		if media.Year > 0 {
			year = fmt.Sprintf(" (%d)", media.Year)
		}
		fmt.Printf("✅ Tag \"%s\" removed from \"%s%s\"\n", tag, media.Title, year)
	},
}

// movie tag list [id]
var tagListCmd = &cobra.Command{
	Use:   "list [id]",
	Short: "List tags (for a media item or all)",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		d, err := db.Open()
		if err != nil {
			errlog.Error("Database error: %v", err)
			return
		}
		defer d.Close()

		if len(args) == 1 {
			listTagsForMedia(d, args[0])
			return
		}
		listAllTags(d)
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)
	tagCmd.AddCommand(tagAddCmd)
	tagCmd.AddCommand(tagRemoveCmd)
	tagCmd.AddCommand(tagListCmd)
}

func listTagsForMedia(d *db.DB, idArg string) {
	id, err := strconv.Atoi(idArg)
	if err != nil {
		errlog.Error("Invalid ID: %s", idArg)
		return
	}
	media, err := d.GetMediaByID(int64(id))
	if err != nil || media == nil {
		errlog.Error("Media not found with ID %d", id)
		return
	}
	tags, err := d.GetTagsByMediaID(id)
	if err != nil {
		errlog.Error("Error reading tags: %v", err)
		return
	}
	year := formatYearSuffix(media.Year)
	if len(tags) == 0 {
		fmt.Printf("📭 No tags for \"%s%s\"\n", media.Title, year)
		return
	}
	fmt.Printf("🏷️  Tags for \"%s%s\":\n", media.Title, year)
	for _, t := range tags {
		fmt.Printf("  • %s\n", t)
	}
}

func listAllTags(d *db.DB) {
	tagCounts, err := d.GetAllTagCounts()
	if err != nil {
		errlog.Error("Error reading tags: %v", err)
		return
	}
	if len(tagCounts) == 0 {
		fmt.Println("📭 No tags in library")
		return
	}
	fmt.Println("🏷️  All tags:")
	for _, tc := range tagCounts {
		fmt.Printf("  %s (%d)\n", tc.Tag, tc.Count)
	}
}
