package guard

import "testing"

func TestCheckCitations_ValidCitationPasses(t *testing.T) {
	issues := CheckCitations("Loopers 아키텍처는 4계층이다 [1].", 2)
	if len(issues) != 0 {
		t.Fatalf("유효 인용인데 이슈 발생: %+v", issues)
	}
}

func TestCheckCitations_UncitedClaimFlagged(t *testing.T) {
	issues := CheckCitations("지구는 둥글다.", 2)
	if len(issues) != 1 || issues[0].Kind != "uncited_claim" {
		t.Fatalf("uncited_claim 을 잡지 못함: %+v", issues)
	}
}

func TestCheckCitations_InvalidCitationFlagged(t *testing.T) {
	issues := CheckCitations("존재하지 않는 근거를 인용한다 [9].", 2)
	if len(issues) != 1 || issues[0].Kind != "invalid_citation" {
		t.Fatalf("invalid_citation 을 잡지 못함: %+v", issues)
	}
}

func TestCheckCitations_ShortFragmentSkipped(t *testing.T) {
	// 3 토큰 미만 조각은 사실 주장으로 보지 않는다.
	if issues := CheckCitations("네.", 1); len(issues) != 0 {
		t.Fatalf("짧은 조각을 이슈로 잡음: %+v", issues)
	}
}
