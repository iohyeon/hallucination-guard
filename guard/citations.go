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
//
// 유효성 판정은 인용 번호가 실제 Evidence.ID 집합의 멤버인지로 한다. Evidence.ID
// 는 임의값이 허용되므로(1..N 연속을 가정하지 않는다) 개수 범위 검사는 비연속
// ID에서 오탐/미탐을 낸다. 그래서 개수가 아니라 실제 ID 집합을 대조한다.
func CheckCitations(answer string, evidence []Evidence) []CitationIssue {
	validIDs := make(map[int]struct{}, len(evidence))
	for _, e := range evidence {
		validIDs[e.ID] = struct{}{}
	}

	var issues []CitationIssue
	for _, raw := range sentenceSplit.Split(answer, -1) {
		s := strings.TrimSpace(raw)
		if len(tokenize(s)) < 2 { // 1토큰 이하 조각은 사실 주장으로 보지 않고 건너뛴다
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
			if _, ok := validIDs[n]; !ok {
				issues = append(issues, CitationIssue{
					Sentence: s, Kind: "invalid_citation",
					Detail: "존재하지 않는 근거 번호 [" + m[1] + "] 인용",
				})
			}
		}
	}
	return issues
}
