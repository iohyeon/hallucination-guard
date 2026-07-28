// Package guard 는 할루시네이션 다층 방어를 구현한다.
// 방어를 생성 파이프라인의 위치로 나눈다:
//
//	예방(그라운딩 게이트) → 생성 → 검출(인용 규칙 · groundedness · judge · 자기검증)
//	→ 게이트(answer / review / abstain)
//
// 각 층은 서로 다른 실패를 잡고, 한 층이 놓쳐도 다음 층이 받친다.
// 모델 출력을 신뢰하지 않고 실측(근거)에 대조해 통과시킨다.
package guard

// Evidence 는 답이 벗어나선 안 되는, 검색된 근거 조각 하나다.
type Evidence struct {
	ID   int    // 인용 번호. 답에서 [ID] 로 참조된다. 관례상 1부터지만 임의값·비연속 허용.
	Text string
}

// Layer 는 신호/이슈를 만든 방어 층이다.
type Layer string

const (
	LayerGrounding   Layer = "grounding"
	LayerCitation    Layer = "citation"
	LayerJudge       Layer = "judge"
	LayerConsistency Layer = "consistency"
)

// Decision 은 최종 게이트 결과다.
type Decision string

const (
	Answer  Decision = "answer"  // 모든 층 통과
	Review  Decision = "review"  // 사람 검토로 에스컬레이션
	Abstain Decision = "abstain" // 반려: "근거 없음" / 미지원
)

// Signal 은 한 층이 낸 정량 측정값이다.
type Signal struct {
	Layer Layer
	Name  string
	Value float64
	Note  string
}

// CitationIssue 는 인용 층의 규칙 기반 발견이다.
type CitationIssue struct {
	Sentence string
	Kind     string // "uncited_claim" | "invalid_citation"
	Detail   string
}

// Verdict 는 LLM-as-judge 의 groundedness 판정 결과다.
type Verdict struct {
	Grounded    bool     `json:"grounded"`
	Unsupported []string `json:"unsupported"`
	Reason      string   `json:"reason"`
}

// Result 는 한 질의에 대한 파이프라인 전체 출력이다.
type Result struct {
	Query          string
	Answer         string
	Decision       Decision
	Reasons        []string
	Signals        []Signal
	CitationIssues []CitationIssue
	Verdict        *Verdict
}
