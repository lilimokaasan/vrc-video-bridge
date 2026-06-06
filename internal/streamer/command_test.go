package streamer

import (
	"testing"
	"time"
)

func TestParseFFmpegOutTime(t *testing.T) {
	got, ok := parseFFmpegOutTime("out_time_ms=1234567")
	if !ok {
		t.Fatal("expected out_time_ms to parse")
	}
	if got != 1234567*time.Microsecond {
		t.Fatalf("expected 1234567 microseconds, got %s", got)
	}

	if _, ok := parseFFmpegOutTime("progress=continue"); ok {
		t.Fatal("expected non out_time_ms line to be ignored")
	}
	if _, ok := parseFFmpegOutTime("out_time_ms=bad"); ok {
		t.Fatal("expected invalid out_time_ms to be ignored")
	}
}
