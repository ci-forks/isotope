package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// cosmicDefaultFavorites is the dock's list a user starts from.
var cosmicDefaultFavorites = "/usr/share/cosmic/com.system76.CosmicAppList/v1/favorites"

var ronString = regexp.MustCompile(`"((?:[^"\\]|\\.)*)"`)

// ronStrings are the strings of a RON list, in order.
func ronStrings(s string) []string {
	var out []string
	for _, m := range ronString.FindAllStringSubmatch(s, -1) {
		out = append(out, m[1])
	}
	return out
}

// PinToCosmicDock appends appID to the user's dock favorites
// (~/.config/cosmic/com.system76.CosmicAppList/v1/favorites, a RON list).
// A user with no file starts from the system default. An app that is
// already in the list is not added again.
func PinToCosmicDock(home, appID string) error {
	path := filepath.Join(home, ".config", "cosmic", "com.system76.CosmicAppList", "v1", "favorites")
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		b, err = os.ReadFile(cosmicDefaultFavorites)
		if os.IsNotExist(err) {
			b, err = []byte("[]"), nil
		}
	}
	if err != nil {
		return err
	}
	s := string(b)
	for _, id := range ronStrings(s) {
		if id == appID {
			return nil
		}
	}
	end := strings.LastIndex(s, "]")
	if end < 0 {
		s, end = "[]", 1
	}
	head := strings.TrimRight(s[:end], " \t\r\n")
	if !strings.HasSuffix(head, "[") && !strings.HasSuffix(head, ",") {
		head += ","
	}
	out := head + "\n    \"" + appID + "\",\n]\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	// Write a temporary file and rename it over the list, so the dock
	// never reads half a file.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".favorites-*")
	if err != nil {
		return err
	}
	if _, err := tmp.WriteString(out); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}
