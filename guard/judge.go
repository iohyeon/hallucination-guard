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
//
// 스키마 강제: Anthropic API 는 output_config.format(json_schema) 또는 tool
// use 의 strict:true 로 응답을 스키마에 강제할 수 있다(anthropic-sdk-go 지원).
// 다만 이 판정은 Mock·Claude 공용의 제네릭 backend.LLM.Generate(문자열 반환)
// 경계를 지나가므로, 스키마 강제를 배선하려면 이 경계에 output_config 를
// 흘려야 한다. 그 배선은 후속 작업으로 두고, 여기서는 응답을 엄격 파싱·검증해
// 견고화했다(parseVerdict). 스키마를 강제하면 아래 파싱은 보조 방어선이 된다.
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

// parseVerdict 는 응답 텍스트에서 첫 JSON 객체를 엄격하게 파싱한다.
//
// 기존 구현은 첫 '{' 와 마지막 '}' 사이를 잘랐는데, reason 문자열에 '}' 가
// 들어 있거나 JSON 뒤에 산문이 붙으면 범위를 잘못 잡아 파싱이 깨질 수 있었다.
// 여기서는 첫 '{' 부터 json.Decoder 로 정확히 한 개의 JSON 값을 디코드한다.
// Decoder 는 중첩 중괄호와 문자열 내부의 '}' 를 올바르게 처리하고 객체가
// 끝나는 지점에서 멈추므로, 뒤에 붙은 잡음에 영향을 받지 않는다.
//
// 파싱 후 최소 검증을 덧붙인다: (1) 실제 JSON '객체' 여야 하고(배열·스칼라
// 거부), (2) grounded=false 인데 근거(unsupported·reason)를 전혀 대지 못한
// 응답은 신뢰할 수 없는 판정으로 보고 거부한다. 스키마를 서버에서 강제하기
// 전까지 이 검증이 관대 파싱의 빈틈을 메운다.
func parseVerdict(s string) (*Verdict, error) {
	start := strings.Index(s, "{")
	if start < 0 {
		return nil, fmt.Errorf("JSON 객체 없음")
	}

	dec := json.NewDecoder(strings.NewReader(s[start:]))
	var v Verdict
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("JSON 객체 디코드 실패: %w", err)
	}

	// grounded=false 는 반드시 근거(미지원 주장 목록 또는 이유)를 동반해야 한다.
	// 둘 다 비면 판정 신뢰도가 없다고 보고 거부한다(무근거 반려 방지).
	if !v.Grounded && len(v.Unsupported) == 0 && strings.TrimSpace(v.Reason) == "" {
		return nil, fmt.Errorf("grounded=false 인데 근거(unsupported·reason)가 비어 있음")
	}
	return &v, nil
}
