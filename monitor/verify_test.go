package monitor

import (
	"testing"

	"github.com/TheEinshine/open_shine/sysstat"
)

func TestCompare(t *testing.T) {
	cases := []struct {
		val  float64
		op   string
		thr  float64
		want bool
	}{
		{91, "gt", 90, true},
		{90, "gt", 90, false},
		{90, "gte", 90, true},
		{5, "lt", 10, true},
		{10, "lte", 10, true},
		{10, "bogus", 10, false},
	}
	for _, c := range cases {
		if got := compare(c.val, c.op, c.thr); got != c.want {
			t.Errorf("compare(%v,%q,%v)=%v want %v", c.val, c.op, c.thr, got, c.want)
		}
	}
}

func TestMetricValue(t *testing.T) {
	s := sysstat.Stats{CPUPercent: 42, MemUsed: 5, MemTotal: 10, DiskUsed: 2, DiskTotal: 8, Load1: 0.5}
	if v, ok := metricValue(s, "cpu"); !ok || v != 42 {
		t.Errorf("cpu = %v,%v", v, ok)
	}
	if v, ok := metricValue(s, "mem"); !ok || v != 50 {
		t.Errorf("mem = %v,%v want 50", v, ok)
	}
	if v, ok := metricValue(s, "disk"); !ok || v != 25 {
		t.Errorf("disk = %v,%v want 25", v, ok)
	}
	if _, ok := metricValue(s, "nope"); ok {
		t.Error("unknown metric should not be ok")
	}
}
