package main

import (
	"gioui.org/font"

	"github.com/kairos-community/isotope/guikit"
)

// The installer's own text styles; the shared ones are in guikit.
var (
	stDoneTitle = guikit.Style{Face: guikit.FaceTitle, Weight: font.Bold, Size: 38}
	stInstTitle = guikit.Style{Face: guikit.FaceTitle, Weight: font.Bold, Size: 26}
	stAvatar    = guikit.Style{Face: guikit.FaceTitle, Weight: font.Bold, Size: 40}
	stMonoKey   = guikit.Style{Face: guikit.FaceMono, Weight: font.Medium, Size: 20}
	stMonoIndex = guikit.Style{Face: guikit.FaceMono, Weight: font.Medium, Size: 11}
	stMonoField = guikit.Style{Face: guikit.FaceMono, Weight: font.Medium, Size: 18}
)
