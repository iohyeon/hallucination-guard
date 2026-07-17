package guard

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iohyeon/hallucination-guard/backend"
)

// judgeSentinel 은 system 프롬프트에 심어 mock 백엔드가 judge 호출을 구분하게
// 하는 표식이다.
const judgeSentinel = "HALLUGUARD_JUDGE"

// LLMJudge 는 별도 LLM 에게 답의 모든 주장이 근거로 뒷받침되는지 판정시킨다.
// 후검증 2층: 인용 번호는 맞지만 그 조각을 미묘하게 왜곡한 경우처럼 규칙이 못
// 잡는 의미 수준 미충실을 잡는다. 구조화된 Verdict 를 돌려준다.
//
// 신뢰도를 올리려면 생성 모델과 다른 모델로 판정(자기 편향 회피)하는 것이
// 바람직하다. 여기서는 백엔드 하나로 단순화했고, 그 확장은 README 에 적었다.
func LLMJudge(ctx context.Context, llm backend.LLM, evidence []Evidence, answer string) (*Verdict, error) {
	var b strings.Builder
	for _, e := range evidence {
		fmt.Fprintf(&b, "[%d] %s\n", e.ID, e.Text)
	}
	system := judgeSentinel + " 너는 사실 검증기다. 컨텍스트로 뒷받침되지 않는 주장이 하나라도 있으면 grounded=false 로 판정한다. 반드시 JSON만 출력한다."
	user := fmt.Sprintf("컨텍스트:\n%s답변:\n%s\n\n출력 스키마(JSON): {\"grounded\": bool, \"unsupported\": [문자열], \"reason\": 문자열}", b.String(), answer)

	out, err := llm.Generate(ctx, system, user)
	if err != nil {
		return nil, err
	}
	v, err := parseVerdict(out)
	if err != nil {
		return nil, fmt.Errorf("judge 응답 파싱 실패: %w (원문: %q)", err, out)
	}
	return v, nil
}

// parseVerdict 는 응답에서 첫 JSON 객체만 관대하게 뽑아 파싱한다.
// (실무에서는 output_config 구조화 출력으로 스키마를 강제하는 편이 견고하다.)
func parseVerdict(s string) (*Verdict, error) {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end < start {
		return nil, fmt.Errorf("JSON 객체 없음")
	}
	var v Verdict
	if err := json.Unmarshal([]byte(s[start:end+1]), &v); err != nil {
		return nil, err
	}
	return &v, nil
}
