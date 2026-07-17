package guard

import (
	"math"
	"regexp"
	"strings"
)

// wordRe 는 영문·숫자·한글 토큰을 뽑는다.
var wordRe = regexp.MustCompile(`[a-z0-9가-힣]+`)

func tokenize(s string) []string {
	return wordRe.FindAllString(strings.ToLower(s), -1)
}

func termFreq(tokens []string) map[string]float64 {
	tf := make(map[string]float64, len(tokens))
	for _, t := range tokens {
		tf[t]++
	}
	return tf
}

// cosine 은 두 토큰 가방의 어휘 코사인 유사도를 [0,1] 로 돌려준다.
// 임베딩 API 없이 어휘 겹침으로 관련도를 근사한다.
// (참고: Anthropic API 는 임베딩 엔드포인트를 제공하지 않으므로, 그라운딩
//
//	관련도는 로컬 어휘 계산으로 근사한다. 실무에서는 임베딩/ANN 검색으로 대체한다.)
func cosine(a, b map[string]float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	var dot, na, nb float64
	for t, av := range a {
		na += av * av
		if bv, ok := b[t]; ok {
			dot += av * bv
		}
	}
	for _, bv := range b {
		nb += bv * bv
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}
