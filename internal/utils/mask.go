package utils

import "strings"

// MaskName 사용자 이름을 마스킹합니다.
// 예:
//   - 김 -> 김
//   - 이순 -> 이*
//   - 김철수 -> 김*수
//   - 남궁민수 -> 남**수
func MaskName(name string) string {
	runes := []rune(strings.TrimSpace(name))
	n := len(runes)
	if n <= 1 {
		return string(runes)
	}
	if n == 2 {
		return string(runes[0]) + "*"
	}
	if n == 3 {
		return string(runes[0]) + "*" + string(runes[2])
	}
	var sb strings.Builder
	sb.WriteRune(runes[0])
	for i := 1; i < n-1; i++ {
		sb.WriteRune('*')
	}
	sb.WriteRune(runes[n-1])
	return sb.String()
}
