// Package backend 은 가드가 의존하는 최소한의 LLM 표면을 정의한다.
// 가드 로직은 이 인터페이스에만 의존하므로, 오프라인 mock 과 실제 Claude 를
// 코드 변경 없이 교체할 수 있다(의존성 역전).
package backend

import "context"

// LLM 은 system+user 프롬프트를 받아 텍스트를 돌려주는 단일 책임 인터페이스다.
// 구현체:
//   - Mock:   오프라인·결정적. 키 없이 파이프라인/테스트를 돌린다.
//   - Claude: 실제 Anthropic API. 판정과 생성을 모델에 위임한다.
type LLM interface {
	Generate(ctx context.Context, system, user string) (string, error)
	Name() string
}
