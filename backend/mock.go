package backend

import (
	"context"
	"strings"
)

// Mock 은 결정적·오프라인 백엔드다. API 키도 네트워크도 없이 파이프라인과
// 테스트를 돌리기 위한 것으로, 추론하지 않고 고정된(검사 가능한) 출력을 낸다.
// 실제 판단은 Claude 백엔드가 담당한다.
type Mock struct {
	// Answer 가 비어 있지 않으면 생성 프롬프트에 대해 이 값을 그대로 반환한다.
	// 비어 있으면 프롬프트의 근거 [1] 을 인용한 답을 만들어낸다.
	Answer string
	// JudgeGrounded 는 judge 프롬프트에 대한 verdict 를 제어한다.
	JudgeGrounded bool
}

// NewMock 은 기본적으로 grounded 판정을 내리는 mock 을 만든다.
func NewMock() *Mock { return &Mock{JudgeGrounded: true} }

func (m *Mock) Name() string { return "mock" }

// Generate 는 judge 프롬프트와 생성 프롬프트를 sentinel 로 구분해 응답한다.
func (m *Mock) Generate(_ context.Context, system, user string) (string, error) {
	if strings.Contains(system, "HALLUGUARD_JUDGE") {
		if m.JudgeGrounded {
			return `{"grounded": true, "unsupported": [], "reason": "mock: 모든 주장이 근거로 뒷받침됨"}`, nil
		}
		return `{"grounded": false, "unsupported": ["mock 미지원 주장"], "reason": "mock: 근거 없는 주장"}`, nil
	}
	if m.Answer != "" {
		return m.Answer, nil
	}
	return firstEvidenceAsAnswer(user), nil
}

// firstEvidenceAsAnswer 는 첫 근거 줄을 인용한 답으로 되돌려, mock 기본 출력이
// 자명하게 grounded 이고 잘 인용되도록 한다.
func firstEvidenceAsAnswer(prompt string) string {
	for _, line := range strings.Split(prompt, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[1]") {
			return strings.TrimSpace(strings.TrimPrefix(line, "[1]")) + " [1]"
		}
	}
	return "문서에 근거 없음"
}
