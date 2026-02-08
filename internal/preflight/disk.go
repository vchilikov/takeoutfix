package preflight

import (
	"fmt"
)

type SpaceCheck struct {
	AvailableBytes          uint64
	RequiredBytes           uint64
	RequiredWithDeleteBytes uint64
	Enough                  bool
	EnoughWithDelete        bool
}

func CheckDiskSpace(path string, archives []ArchiveIntegrity) (SpaceCheck, error) {
	available, err := availableDisk(path)
	if err != nil {
		return SpaceCheck{}, err
	}

	required, requiredWithDelete := estimateRequiredBytes(archives)

	return SpaceCheck{
		AvailableBytes:          available,
		RequiredBytes:           required,
		RequiredWithDeleteBytes: requiredWithDelete,
		Enough:                  available >= required,
		EnoughWithDelete:        available >= requiredWithDelete,
	}, nil
}

func estimateRequiredBytes(archives []ArchiveIntegrity) (uint64, uint64) {
	var normalBase uint64
	var deleteModePeak int64
	var prefixNetBeforeDelete int64

	for _, archive := range archives {
		if archive.Err != nil {
			continue
		}

		normalBase += archive.UncompressedBytes

		currentPeak := prefixNetBeforeDelete + int64(archive.UncompressedBytes)
		if currentPeak > deleteModePeak {
			deleteModePeak = currentPeak
		}

		prefixNetBeforeDelete += int64(archive.UncompressedBytes) - int64(archive.Archive.SizeBytes)
	}

	if deleteModePeak < 0 {
		deleteModePeak = 0
	}

	return addMargin(normalBase), addMargin(uint64(deleteModePeak))
}

func addMargin(v uint64) uint64 {
	extra := v / 10
	if v%10 != 0 {
		extra++
	}
	if ^uint64(0)-v < extra {
		return ^uint64(0)
	}
	return v + extra
}

func FormatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
