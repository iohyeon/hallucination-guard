package guard

import (
	"regexp"
	"strings"
)

var citationRe = regexp.MustCompile(`\[(\d+)\]`)
var sentenceSplit = regexp.MustCompile(`[.!?。！？\n]+`)

// CheckCitations 는 답의 각 사실 문장이 유효한 근거 번호를 인용하는지 규칙으로
// 검증한다. 후검증 1층(LLM 없음): 존재하지 않는 번호 인용이나 인용 없는 사실
// 문장 같은 명백한 위조를 값싸고 결정적으로 잡는다. 규칙이 못 잡는 의미 수준의
// 왜곡은 LLMJudge(2층)로 넘긴다.
func CheckCitations(answer string, evidenceCount int) []CitationIssue {
	var issues []CitationIssue
	for _, raw := range sentenceSplit.Split(answer, -1) {
		s := strings.TrimSpace(raw)
		if len(tokenize(s)) < 3 { // 조각은 사실 주장으로 보지 않고 건너뛴다
			continue
		}
		nums := citationRe.FindAllStringSubmatch(s, -1)
		if len(nums) == 0 {
			issues = append(issues, CitationIssue{
				Sentence: s, Kind: "uncited_claim",
				Detail: "사실 문장에 근거 인용 [n] 이 없음",
			})
			continue
		}
		for _, m := range nums {
			n := atoi(m[1])
			if n < 1 || n > evidenceCount {
				issues = append(issues, CitationIssue{
					Sentence: s, Kind: "invalid_citation",
					Detail: "존재하지 않는 근거 번호 [" + m[1] + "] 인용",
				})
			}
		}
	}
	return issues
}
