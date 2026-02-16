package files

import "testing"

func TestNormalizeJSONKey(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "classic media json",
			in:   "IMG_0001.JPG.json",
			want: "img_0001",
		},
		{
			name: "plain json",
			in:   "IMG_0001.json",
			want: "img_0001",
		},
		{
			name: "supplemental metadata suffix",
			in:   "IMG_0001.jpg.supplemental-metadata.json",
			want: "img_0001",
		},
		{
			name: "supplemental truncated suffix",
			in:   "IMG_0001.jpg.supplemental-metada.json",
			want: "img_0001",
		},
		{
			name: "multiple dots",
			in:   "PXL.2024.IMG_0001.jpg.supplemental-metadata.json",
			want: "pxl.2024.img_0001",
		},
		{
			name: "truncated supplemental .suppl suffix",
			in:   "IMG_0001.jpg.suppl.json",
			want: "img_0001",
		},
		{
			name: "truncated supplemental .supplemental-met suffix",
			in:   "IMG_0001.jpg.supplemental-met.json",
			want: "img_0001",
		},
		{
			name: "truncated supplemental with dedup number",
			in:   "IMG_0001.jpg.suppl(1).json",
			want: "img_0001",
		},
		{
			name: "double extension with truncated supplemental",
			in:   "VID_0001.mp4.mov.supplemental-m.json",
			want: "vid_0001",
		},
		{
			name: "duplicate index in media stem with supplemental metadata",
			in:   "IMG_0001(0).jpg.supplemental-metadata.json",
			want: "img_0001",
		},
		{
			name: "duplicate index in media stem with truncated supplemental suffix",
			in:   "IMG_0001(12).JPG.supplemental-metada.json",
			want: "img_0001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeJSONKey(tt.in)
			if got != tt.want {
				t.Fatalf("normalizeJSONKey(%q): want %q, got %q", tt.in, tt.want, got)
			}
		})
	}
}

func TestNormalizeMediaLookupKey_Deterministic(t *testing.T) {
	input := "IMG_0001-ABCDE.JPG"
	want := "img_0001"

	got1 := normalizeMediaLookupKey(input)
	got2 := normalizeMediaLookupKey(input)
	if got1 != want || got2 != want || got1 != got2 {
		t.Fatalf("expected deterministic normalized media key %q, got %q and %q", want, got1, got2)
	}
}
