package website

import (
	"fmt"
	"strings"

	"github.com/karty-game/karty-packages/internal/catalog"
)

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func installCommand(name string, version catalog.Version) string {
	filename := name + "-" + version.Version + ".kpkg"
	return "curl --fail --location --proto '=https' --proto-redir '=https' \\\n  --output " + shellQuote(filename) + " \\\n  " + shellQuote(version.Archive) + " &&\n" +
		"printf '%s  %s\\n' " + shellQuote(version.SHA256) + " " + shellQuote(filename) + " | sha256sum --check - &&\n" +
		"karty package import " + shellQuote(filename)
}

func markdownText(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "`", "\\`", "*", "\\*", "_", "\\_", "[", "\\[", "]", "\\]", "<", "&lt;", ">", "&gt;", "\r", " ", "\n", " ", "|", "\\|")
	return replacer.Replace(value)
}

func markdownLink(label, address string) string {
	// Angle destinations preserve parentheses, spaces and Markdown delimiters.
	address = strings.NewReplacer("<", "%3C", ">", "%3E").Replace(address)
	return "[" + markdownText(label) + "](<" + address + ">)"
}

func indexMarkdown(data pageData) string {
	var result strings.Builder
	result.WriteString("# Karty Packages\n\nReusable code and assets for your next Karty game.\n\n")
	for _, entry := range data.Packages {
		fmt.Fprintf(&result, "## [%s](packages/%s.md)\n\n%s\n\n", markdownText(entry.DisplayName), entry.Name, markdownText(entry.Description))
		if len(entry.Tags) > 0 {
			fmt.Fprintf(&result, "Tags: %s.\n\n", strings.Join(entry.Tags, ", "))
		}
		if entry.Latest != nil {
			fmt.Fprintf(&result, "Latest: %s.\n\n", *entry.Latest)
		} else {
			result.WriteString("No published release yet.\n\n")
		}
	}
	return result.String()
}

func packageMarkdown(entry packageView) string {
	var result strings.Builder
	fmt.Fprintf(&result, "# %s\n\n%s\n\n", markdownText(entry.DisplayName), markdownText(entry.Description))
	if entry.Preview != "" {
		fmt.Fprintf(&result, "!%s\n\n", markdownLink(entry.DisplayName+" preview", entry.Preview))
	}
	fmt.Fprintf(&result, "Repository: %s.\n\n%s · %s\n\n", markdownText(entry.Repository), markdownLink("Read the README", entry.Readme), markdownLink("Browse source", entry.Source))
	if len(entry.Tags) > 0 {
		fmt.Fprintf(&result, "Tags: %s.\n\n", strings.Join(entry.Tags, ", "))
	}
	result.WriteString("## Install\n\n")
	if entry.Latest == nil {
		result.WriteString("No published release yet. Installation instructions will appear after a release is published.\n\n")
	} else {
		fmt.Fprintf(&result, "Run from your Karty project directory. Downloads are checked against the published SHA-256 before import.\n\n```sh\n%s\n```\n\nFor an existing package, add `--replace` to the import command.\n\n", entry.Command)
	}
	result.WriteString("## Versions\n\n")
	if len(entry.ReleaseVersions) == 0 {
		result.WriteString("No published versions.\n")
	}
	for _, version := range entry.ReleaseVersions {
		fmt.Fprintf(&result, "### %s\n\n%s · %s\n\nSHA-256: `%s`\n\n```sh\n%s\n```\n\n", version.Version.Version, markdownLink("Release notes", version.ReleaseURL), markdownLink("Download archive", version.Archive), version.SHA256, version.Command)
	}
	return result.String()
}
