package domain

import "testing"

func TestParsePriceBRL(t *testing.T) {
	cases := []struct {
		raw  string
		want int64
	}{
		{"R$                143.12", 1431200},
		{"R$      1,465.57", 14655700},
		{"0", 0},
		{"14.31", 143100},
		{"1.465,57", 14655700},
		{"12.52345", 125235}, // half-up from 12.52345 → 12.5235? wait 5th digit 5 → round up 12.5235? places: 5234|5 → 5235 → 12.5235 scaled = 125235
	}
	for _, tc := range cases {
		got, err := ParsePriceBRL(tc.raw)
		if err != nil {
			t.Fatalf("%q: %v", tc.raw, err)
		}
		if got != tc.want {
			t.Fatalf("%q = %d want %d", tc.raw, got, tc.want)
		}
	}
}

func TestFormatMoney(t *testing.T) {
	if got := FormatMoney(14655700); got != "1465.5700" {
		t.Fatalf("got %s", got)
	}
}
