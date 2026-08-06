package guard

import (
	"context"
	"fmt"
	"strings"

	"github.com/iohyeon/hallucination-guard/backend"
)

// Config 는 방어 임계값을 조정한다. Default() 가 합리적 기본값을 준다.
type Config struct {
	MinEvidenceSimilarity float64 // 검색 시 이 값 미만이면 생성 전에 기권
	MinGroundedness       float64 // 답-근거 어휘 커버리지 하한
	MaxEntropy            float64 // 이 값 초과면 검토로 에스컬레이션
	ConsistencySamples    int     // 엔트로피 표본 수. 2 미만이면 비활성
	UseJudge              bool    // LLM-as-judge 층 실행 여부
	CitationStrict        bool    // 인용 없는 사실 문장을 검토로 올릴지
}

// Default 는 데모/테스트에 적당한 기본 설정이다.
func Default() Config {
	return Config{
		MinEvidenceSimilarity: 0.08,
		MinGroundedness:       0.35,
		MaxEntropy:            0.5,
		ConsistencySamples:    0,
		UseJudge:              true,
		CitationStrict:        true,
	}
}

// Pipeline 은 LLM 백엔드 둘레에 방어 층들을 배선한다.
type Pipeline struct {
	LLM backend.LLM
	Cfg Config
}

// New 는 파이프라인을 만든다.
func New(llm backend.LLM, cfg Config) *Pipeline { return &Pipeline{LLM: llm, Cfg: cfg} }

// Run 은 한 질의에 대해 예방 → 생성 → 검출 → 게이트를 순서대로 실행한다.
// 싼 층을 앞에 두고 명백한 케이스를 먼저 쳐낸다. 구체적으로 두 지점에서
// 조기 단락한다: (1) 관련 근거가 없으면 생성 자체를 건너뛰고, (2) 규칙만으로
// 확정 반려가 가능한 케이스(허위 인용 등)는 비싼 LLM 층(엔트로피 · judge)을
// 호출하기 전에 즉시 기권해 judge 호출 수를 줄인다.
func (p *Pipeline) Run(ctx context.Context, query string, evidence []Evidence) (*Result, error) {
	r := &Result{Query: query, Decision: Answer}

	// 예방: 그라운딩 게이트. 관련 근거가 없으면 생성하지 않고 기권한다.
	sim, bestID := MaxSimilarity(query, evidence)
	r.Signals = append(r.Signals, Signal{
		Layer: LayerGrounding, Name: "max_similarity", Value: sim,
		Note: fmt.Sprintf("최고 유사 근거 [%d]", bestID),
	})
	if sim < p.Cfg.MinEvidenceSimilarity {
		r.Decision = Abstain
		r.Reasons = append(r.Reasons, "근거 부족: 관련 근거를 찾지 못해 생성을 건너뜀")
		r.Answer = "문서에 근거 없음"
		return r, nil
	}

	// 생성: 근거에 그라운딩하고 인용을 강제한다.
	prompt := buildPrompt(query, evidence)
	answer, err := p.LLM.Generate(ctx, groundedSystem, prompt)
	if err != nil {
		return nil, err
	}
	r.Answer = answer

	// 검출(싼 규칙): 인용 검증.
	r.CitationIssues = CheckCitations(answer, evidence)

	// 검출(싼 신호): 어휘 groundedness. LLM 호출이 없는 무료 신호이므로 조기
	// 단락 판정 전에 계산해 결과에 남긴다.
	g := LexicalGroundedness(answer, evidence)
	r.Signals = append(r.Signals, Signal{Layer: LayerGrounding, Name: "lexical_groundedness", Value: g})

	// 조기 단락(비용 절감): 규칙만으로 확정 반려가 가능한 케이스는 비싼 LLM 층
	// (시맨틱 엔트로피 · judge)을 호출하기 전에 즉시 기권한다.
	//
	// invalid_citation(존재하지 않는 근거 번호 인용)은 규칙 층이 결정적으로 잡는
	// 명백한 위조이고, 최종 게이트에서도 어차피 abstain 으로 접힌다. 뒤 층을 더
	// 돌려도 abstain(심각도 2)보다 높은 결정은 나올 수 없으므로, 여기서 판정을
	// 확정하고 남은 비싼 층을 건너뛰어 judge 호출 수를 줄인다.
	//
	// 반대로 "낮은 groundedness" 나 "uncited_claim" 만으로는 조기 단락하지 않는다.
	//   - 낮은 어휘 커버리지는 근거를 정상적으로 재진술(패러프레이즈)한 답에서도
	//     흔히 나온다. 이것만으로 judge 를 생략하면 정상 답을 잘못 막게 되므로,
	//     낮은 groundedness 는 review 로만 올리고(judge 생략 근거로 쓰지 않음)
	//     judge 는 그대로 호출해 의미 수준 충실성을 판정하게 둔다.
	//   - uncited_claim 역시 확정 반려가 아니라 review(또는 CitationStrict=false
	//     면 무시)라서, 조기 단락의 근거로 삼지 않는다.
	if hasInvalidCitation(r.CitationIssues) {
		r.applyCitationGate(p.Cfg)
		return r, nil
	}

	// 검출(선택, N 표본): 시맨틱 엔트로피.
	if p.Cfg.ConsistencySamples >= 2 {
		ent, _, err := SemanticEntropy(ctx, p.LLM, groundedSystem, prompt, p.Cfg.ConsistencySamples)
		if err != nil {
			return nil, err
		}
		r.Signals = append(r.Signals, Signal{Layer: LayerConsistency, Name: "semantic_entropy", Value: ent})
		if ent > p.Cfg.MaxEntropy {
			r.escalate(Review, "표본 간 답이 갈림(불확실성 높음)")
		}
	}

	// 검출(비쌈, 선택): LLM-as-judge.
	if p.Cfg.UseJudge {
		v, err := LLMJudge(ctx, p.LLM, evidence, answer)
		if err != nil {
			return nil, err
		}
		r.Verdict = v
		if !v.Grounded {
			r.escalate(Abstain, "judge: 근거로 뒷받침되지 않는 주장 존재")
		}
	}

	// 게이트: 규칙 발견을 최종 결정에 접는다.
	if g < p.Cfg.MinGroundedness {
		r.escalate(Review, fmt.Sprintf("근거 커버리지 낮음(%.2f)", g))
	}
	r.applyCitationGate(p.Cfg)
	return r, nil
}

// hasInvalidCitation 은 인용 이슈 중 invalid_citation(허위 인용)이 하나라도
// 있는지 본다. 조기 단락 판정에 쓰인다.
func hasInvalidCitation(issues []CitationIssue) bool {
	for _, is := range issues {
		if is.Kind == "invalid_citation" {
			return true
		}
	}
	return false
}

// applyCitationGate 는 인용 층의 규칙 발견을 최종 결정에 접는다. 조기 단락
// 경로와 정상 경로 양쪽에서 같은 규칙을 쓰도록 헬퍼로 뽑았다.
//   - invalid_citation → abstain(확정 반려)
//   - uncited_claim    → review(단, CitationStrict 일 때만)
func (r *Result) applyCitationGate(cfg Config) {
	for _, is := range r.CitationIssues {
		switch is.Kind {
		case "invalid_citation":
			r.escalate(Abstain, "허위 인용 발견")
		case "uncited_claim":
			if cfg.CitationStrict {
				r.escalate(Review, "근거 없는 사실 문장")
			}
		}
	}
}

// escalate 는 결정 심각도를 올리고(answer < review < abstain) 이유를 남긴다.
func (r *Result) escalate(d Decision, reason string) {
	r.Reasons = append(r.Reasons, reason)
	if severity(d) > severity(r.Decision) {
		r.Decision = d
	}
}

func severity(d Decision) int {
	switch d {
	case Abstain:
		return 2
	case Review:
		return 1
	default:
		return 0
	}
}

const groundedSystem = "너는 주어진 근거 안에서만 답한다. 각 사실 문장 끝에 근거 번호를 [n] 형식으로 붙인다. 근거에 없는 내용은 쓰지 않고, 없으면 '문서에 근거 없음'이라고 답한다."

func buildPrompt(query string, evidence []Evidence) string {
	var b strings.Builder
	b.WriteString("근거:\n")
	for _, e := range evidence {
		fmt.Fprintf(&b, "[%d] %s\n", e.ID, e.Text)
	}
	fmt.Fprintf(&b, "\n질문: %s\n", query)
	return b.String()
}
