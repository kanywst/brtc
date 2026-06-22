package ui

import (
	"strings"
	"testing"
)

func sampleMatrix() []MatrixRow {
	return []MatrixRow{
		{Profile: "aws-p5.48xlarge", Name: "AWS", HashRate: 2500000, TimeToCrackSec: 3600, CostUSD: 40, CostPerHourUSD: 40},
		{Profile: "mac-m3", Name: "Apple M3", HashRate: 5500, TimeToCrackSec: 7200, CostUSD: 0, CostPerHourUSD: 0},
	}
}

func TestRenderMatrixTable(t *testing.T) {
	var buf strings.Builder
	if err := renderMatrixTable(&buf, sampleMatrix()); err != nil {
		t.Fatalf("renderMatrixTable failed: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "HARDWARE") || !strings.Contains(out, "aws-p5.48xlarge") {
		t.Errorf("matrix table missing headers/rows:\n%s", out)
	}
	// Owned hardware (cost_per_hour 0) is shown as "owned", not "$0.00".
	if !strings.Contains(out, "owned") {
		t.Errorf("owned hardware should render as \"owned\":\n%s", out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("matrix table should be ANSI-free:\n%q", out)
	}
}
