package guard

import "testing"

func TestCheckCitations_ValidCitationPasses(t *testing.T) {
	ev := []Evidence{{ID: 1, Text: "a"}, {ID: 2, Text: "b"}}
	issues := CheckCitations("Loopers 아키텍처는 4계층이다 [1].", ev)
	if len(issues) != 0 {
		t.Fatalf("유효 인용인데 이슈 발생: %+v", issues)
	}
}

func TestCheckCitations_UncitedClaimFlagged(t *testing.T) {
	ev := []Evidence{{ID: 1, Text: "a"}, {ID: 2, Text: "b"}}
	issues := CheckCitations("지구는 둥글다.", ev)
	if len(issues) != 1 || issues[0].Kind != "uncited_claim" {
		t.Fatalf("uncited_claim 을 잡지 못함: %+v", issues)
	}
}

func TestCheckCitations_InvalidCitationFlagged(t *testing.T) {
	ev := []Evidence{{ID: 1, Text: "a"}, {ID: 2, Text: "b"}}
	issues := CheckCitations("존재하지 않는 근거를 인용한다 [9].", ev)
	if len(issues) != 1 || issues[0].Kind != "invalid_citation" {
		t.Fatalf("invalid_citation 을 잡지 못함: %+v", issues)
	}
}

func TestCheckCitations_ShortFragmentSkipped(t *testing.T) {
	// 1 토큰 이하 조각은 사실 주장으로 보지 않는다.
	ev := []Evidence{{ID: 1, Text: "a"}}
	if issues := CheckCitations("네.", ev); len(issues) != 0 {
		t.Fatalf("짧은 조각을 이슈로 잡음: %+v", issues)
	}
}

// 비연속 ID(예: {1,5})에서 개수 범위 검사(n<1 || n>N)는 오탐/미탐을 낸다.
// 유효성은 실제 Evidence.ID 집합 멤버십으로 판정해야 한다.
func TestCheckCitations_NonContiguousIDs(t *testing.T) {
	ev := []Evidence{{ID: 1, Text: "a"}, {ID: 5, Text: "b"}}

	// (a) [5] 는 ID 집합의 멤버이므로 유효하게 통과해야 한다.
	//     개수 범위 검사였다면 5 > 2 로 오탐 반려됐을 것이다.
	if issues := CheckCitations("근거 다섯을 인용한다 [5].", ev); len(issues) != 0 {
		t.Fatalf("유효한 비연속 ID 인용 [5] 를 허위 반려함: %+v", issues)
	}

	// (b) [2] 는 ID 집합에 없으므로 invalid_citation 으로 잡혀야 한다.
	//     개수 범위 검사였다면 2 <= 2 로 미탐 통과됐을 것이다.
	issues := CheckCitations("존재하지 않는 근거 둘을 인용한다 [2].", ev)
	if len(issues) != 1 || issues[0].Kind != "invalid_citation" {
		t.Fatalf("존재하지 않는 ID 인용 [2] 를 잡지 못함: %+v", issues)
	}
}
