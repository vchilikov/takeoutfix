package files

import (
	"strings"
	"testing"
)

func TestGetJsonFile(t *testing.T) {
	longMedia := strings.Repeat("a", 50) + ".jpg"
	longMediaWithSuffix := strings.Repeat("b", 50) + "(2).jpg"

	tests := []struct {
		name      string
		mediaFile string
		jsonFiles map[string]struct{}
		want      string
		wantErr   bool
	}{
		{
			name:      "exact match",
			mediaFile: "IMG_0001.jpg",
			jsonFiles: map[string]struct{}{"IMG_0001.jpg.json": {}},
			want:      "IMG_0001.jpg.json",
		},
		{
			name:      "supplemental metadata json",
			mediaFile: "IMG_0001.jpg",
			jsonFiles: map[string]struct{}{"IMG_0001.jpg.supplemental-metadata.json": {}},
			want:      "IMG_0001.jpg.supplemental-metadata.json",
		},
		{
			name:      "supplemental metada truncated suffix json",
			mediaFile: "IMG_0001.jpg",
			jsonFiles: map[string]struct{}{"IMG_0001.jpg.supplemental-metada.json": {}},
			want:      "IMG_0001.jpg.supplemental-metada.json",
		},
		{
			name:      "supplemental metadata without media extension in json stem",
			mediaFile: "IMG_0001.jpg",
			jsonFiles: map[string]struct{}{"IMG_0001.supplemental-metadata.json": {}},
			want:      "IMG_0001.supplemental-metadata.json",
		},
		{
			name:      "supplemental metadata with mixed case extension",
			mediaFile: "IMG_0001.jpg",
			jsonFiles: map[string]struct{}{"IMG_0001.JPG.SUPPLEMENTAL-METADATA.JSON": {}},
			want:      "IMG_0001.JPG.SUPPLEMENTAL-METADATA.JSON",
		},
		{
			name:      "edited file fallback",
			mediaFile: "IMG_0001-edited.jpg",
			jsonFiles: map[string]struct{}{"IMG_0001.jpg.json": {}},
			want:      "IMG_0001.jpg.json",
		},
		{
			name:      "number suffix handling",
			mediaFile: "IMG_0001(1).jpg",
			jsonFiles: map[string]struct{}{"IMG_0001.jpg(1).json": {}},
			want:      "IMG_0001.jpg(1).json",
		},
		{
			name:      "long filename truncation",
			mediaFile: longMedia,
			jsonFiles: map[string]struct{}{longMedia[:46] + ".json": {}},
			want:      longMedia[:46] + ".json",
		},
		{
			name:      "long filename with number suffix truncation",
			mediaFile: longMediaWithSuffix,
			jsonFiles: map[string]struct{}{longMediaWithSuffix[:46] + "(2).json": {}},
			want:      longMediaWithSuffix[:46] + "(2).json",
		},
		{
			name:      "mp4 fallback to upper-case image extension json",
			mediaFile: "PXL_001.mp4",
			jsonFiles: map[string]struct{}{"PXL_001.JPG.json": {}},
			want:      "PXL_001.JPG.json",
		},
		{
			name:      "idempotent after extension rename",
			mediaFile: "IMG_0001.png",
			jsonFiles: map[string]struct{}{"IMG_0001.jpg.json": {}},
			want:      "IMG_0001.jpg.json",
		},
		{
			name:      "idempotent after random suffix rename",
			mediaFile: "IMG_0001-abcde.png",
			jsonFiles: map[string]struct{}{"IMG_0001.jpg.json": {}},
			want:      "IMG_0001.jpg.json",
		},
		{
			name:      "ambiguous basename fallback returns error",
			mediaFile: "IMG_0001.png",
			jsonFiles: map[string]struct{}{"IMG_0001.jpg.json": {}, "IMG_0001.heic.json": {}},
			wantErr:   true,
		},
		{
			name:      "truncated supplemental .suppl suffix",
			mediaFile: "IMG_0001.jpg",
			jsonFiles: map[string]struct{}{"IMG_0001.jpg.suppl.json": {}},
			want:      "IMG_0001.jpg.suppl.json",
		},
		{
			name:      "truncated supplemental .sup suffix",
			mediaFile: "IMG_0001.jpg",
			jsonFiles: map[string]struct{}{"IMG_0001.jpg.sup.json": {}},
			want:      "IMG_0001.jpg.sup.json",
		},
		{
			name:      "truncated supplemental .supp suffix",
			mediaFile: "IMG_0001.jpg",
			jsonFiles: map[string]struct{}{"IMG_0001.jpg.supp.json": {}},
			want:      "IMG_0001.jpg.supp.json",
		},
		{
			name:      "truncated supplemental .supplemental-met suffix",
			mediaFile: "IMG_0001.jpg",
			jsonFiles: map[string]struct{}{"IMG_0001.jpg.supplemental-met.json": {}},
			want:      "IMG_0001.jpg.supplemental-met.json",
		},
		{
			name:      "truncated supplemental with dedup number",
			mediaFile: "IMG_0001.jpg",
			jsonFiles: map[string]struct{}{"IMG_0001.jpg.suppl(1).json": {}},
			want:      "IMG_0001.jpg.suppl(1).json",
		},
		{
			name:      "double extension mp4.mov with truncated supplemental",
			mediaFile: "VID_0001.mp4.mov",
			jsonFiles: map[string]struct{}{"VID_0001.mp4.mov.supplemental-m.json": {}},
			want:      "VID_0001.mp4.mov.supplemental-m.json",
		},
		{
			name:      "minimal truncated supplemental .s suffix",
			mediaFile: "IMG_0001.jpg",
			jsonFiles: map[string]struct{}{"IMG_0001.jpg.s.json": {}},
			want:      "IMG_0001.jpg.s.json",
		},
		{
			name:      "not found",
			mediaFile: "missing.jpg",
			jsonFiles: map[string]struct{}{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getJsonFile(tt.mediaFile, tt.jsonFiles)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("want %q, got %q", tt.want, got)
			}
		})
	}
}
