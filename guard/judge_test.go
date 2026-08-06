package guard

import "testing"

// reason 문자열에 '}' 가 들어가도(첫 '{' ~ 마지막 '}' 자르기의 약점) 정확히
// 첫 JSON 객체만 디코드해야 한다.
func TestParseVerdict_BraceInStringField(t *testing.T) {
	v, err := parseVerdict(`{"grounded": true, "unsupported": [], "reason": "코드 조각 f(){} 는 근거에 있음"}`)
	if err != nil {
		t.Fatalf("정상 JSON 파싱 실패: %v", err)
	}
	if !v.Grounded {
		t.Fatalf("grounded=true 여야 함: %+v", v)
	}
}

// JSON 뒤에 산문이 붙어도 첫 객체에서 멈춰 파싱해야 한다.
func TestParseVerdict_TrailingProse(t *testing.T) {
	v, err := parseVerdict(`설명: {"grounded": false, "unsupported": ["x"], "reason": "근거 없음"} 이상입니다.`)
	if err != nil {
		t.Fatalf("후행 산문이 있는 JSON 파싱 실패: %v", err)
	}
	if v.Grounded {
		t.Fatalf("grounded=false 여야 함: %+v", v)
	}
}

// JSON 객체가 없으면 에러여야 한다.
func TestParseVerdict_NoObject(t *testing.T) {
	if _, err := parseVerdict("판정 불가"); err == nil {
		t.Fatalf("JSON 객체가 없는데 에러가 아님")
	}
}

// grounded=false 인데 근거를 전혀 대지 못한 판정은 신뢰할 수 없어 거부한다.
func TestParseVerdict_UngroundedWithoutEvidenceRejected(t *testing.T) {
	if _, err := parseVerdict(`{"grounded": false, "unsupported": [], "reason": ""}`); err == nil {
		t.Fatalf("무근거 반려 판정은 거부돼야 함")
	}
}
