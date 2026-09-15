package utils

import "testing"

func TestMaskName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"빈 문자열", "", ""},
		{"한 글자", "김", "김"},
		{"두 글자", "이순", "이*"},
		{"세 글자", "김철수", "김*수"},
		{"세 글자 2", "홍길동", "홍*동"},
		{"네 글자", "남궁민수", "남**수"},
		{"다섯 글자", "스쿠잇짱짱", "스***짱"},
		{"영문 네 글자", "John", "J**n"},
		{"영문 다섯 글자", "Alice", "A***e"},
		{"공백 포함", "  박수철  ", "박*철"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskName(tt.input)
			if got != tt.expected {
				t.Errorf("MaskName(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}
