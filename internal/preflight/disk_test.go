package preflight

import "testing"

func TestCheckDiskSpaceSanity(t *testing.T) {
	archives := []ArchiveIntegrity{
		{
			Archive:           ZipArchive{SizeBytes: 512 * 1024},
			UncompressedBytes: 1024 * 1024,
		},
	}

	res, err := CheckDiskSpace(".", archives)
	if err != nil {
		t.Fatalf("CheckDiskSpace error: %v", err)
	}
	if res.RequiredBytes <= 1024*1024 {
		t.Fatalf("expected required with margin > base")
	}
	if res.RequiredWithDeleteBytes > res.RequiredBytes {
		t.Fatalf("expected delete-mode requirement <= normal requirement")
	}
}

func TestEstimateRequiredBytes_UsesPeakForDeleteMode(t *testing.T) {
	archives := []ArchiveIntegrity{
		{
			Archive:           ZipArchive{SizeBytes: 10},
			UncompressedBytes: 100,
		},
		{
			Archive:           ZipArchive{SizeBytes: 95},
			UncompressedBytes: 100,
		},
	}

	required, requiredWithDelete := estimateRequiredBytes(archives)
	if required != 220 {
		t.Fatalf("required mismatch: want 220, got %d", required)
	}
	if requiredWithDelete != 209 {
		t.Fatalf("requiredWithDelete mismatch: want 209, got %d", requiredWithDelete)
	}
}

func TestEstimateRequiredBytes_IgnoresCorruptArchives(t *testing.T) {
	archives := []ArchiveIntegrity{
		{
			Archive:           ZipArchive{SizeBytes: 50},
			UncompressedBytes: 100,
		},
		{
			Archive: ZipArchive{SizeBytes: 50},
			Err:     errSyntheticDiskTest{},
		},
	}

	required, requiredWithDelete := estimateRequiredBytes(archives)
	if required != 110 {
		t.Fatalf("required mismatch: want 110, got %d", required)
	}
	if requiredWithDelete != 110 {
		t.Fatalf("requiredWithDelete mismatch: want 110, got %d", requiredWithDelete)
	}
}

func TestFormatBytes(t *testing.T) {
	if got := FormatBytes(1536); got == "1536 B" {
		t.Fatalf("expected human-readable unit, got %q", got)
	}
}

type errSyntheticDiskTest struct{}

func (errSyntheticDiskTest) Error() string {
	return "synthetic"
}
