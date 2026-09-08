package main

import "testing"

func TestMaskName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"김", "김"},
		{"이순", "이*"},
		{"김철수", "김*수"},
		{"홍길동", "홍*동"},
		{"남궁민수", "남**수"},
		{"스쿠잇짱짱", "스***짱"},
		{"John", "J**n"},
		{"Alice", "A***e"},
		{"  박수철  ", "박*철"},
	}

	for _, tt := range tests {
		got := maskName(tt.input)
		if got != tt.expected {
			t.Errorf("maskName(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}
