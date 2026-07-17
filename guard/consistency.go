package guard

import (
	"context"
	"math"
	"strings"

	"github.com/iohyeon/hallucination-guard/backend"
)

// SemanticEntropy 는 같은 질문을 n 번 샘플링해 답들이 얼마나 갈리는지를
// 정규화 섀넌 엔트로피 [0,1] 로 잰다. 값이 높으면 모델이 갈팡질팡한다는
// 불확실성 신호이고, 기권의 근거가 된다.
//
// 주의: 여기서는 답을 정규화된 "문자열" 단위로 군집한다. 이는 진짜 시맨틱
// 엔트로피(의미 동치끼리 함의 기준으로 군집)의 어휘 근사다. 같은 뜻을 다르게
// 말한 패러프레이즈를 다른 답으로 세어 불확실성을 과대평가할 수 있다. 정식
// 구현은 함의 판정 모델로 의미 군집을 만들어야 한다(README 참조).
func SemanticEntropy(ctx context.Context, llm backend.LLM, system, user string, n int) (float64, []string, error) {
	if n < 2 {
		n = 2
	}
	clusters := map[string]int{}
	samples := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out, err := llm.Generate(ctx, system, user)
		if err != nil {
			return 0, nil, err
		}
		samples = append(samples, out)
		clusters[normalize(out)]++
	}
	var h float64
	for _, c := range clusters {
		p := float64(c) / float64(n)
		h -= p * math.Log2(p)
	}
	maxH := math.Log2(float64(n))
	if maxH == 0 {
		return 0, samples, nil
	}
	return h / maxH, samples, nil
}

func normalize(s string) string {
	return strings.Join(tokenize(s), " ")
}
