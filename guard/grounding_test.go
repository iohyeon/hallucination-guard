package guard

import "testing"

func TestMaxSimilarity_RelevantEvidenceScoresHigh(t *testing.T) {
	ev := []Evidence{
		{ID: 1, Text: "Loopers 아키텍처는 4계층으로 구성된다"},
		{ID: 2, Text: "고양이는 포유류다"},
	}
	sim, id := MaxSimilarity("Loopers 아키텍처 계층", ev)
	if id != 1 {
		t.Fatalf("가장 관련된 근거 [1] 을 못 고름: id=%d", id)
	}
	if sim <= 0 {
		t.Fatalf("관련 근거인데 유사도 0: %f", sim)
	}
}

func TestMaxSimilarity_UnrelatedScoresZero(t *testing.T) {
	ev := []Evidence{{ID: 1, Text: "고양이는 포유류다"}}
	sim, _ := MaxSimilarity("quantum entanglement", ev)
	if sim != 0 {
		t.Fatalf("무관한데 유사도가 0이 아님: %f", sim)
	}
}

func TestLexicalGroundedness_CoveredVsExtrinsic(t *testing.T) {
	ev := []Evidence{{ID: 1, Text: "아키텍처는 4계층으로 구성된다"}}
	covered := LexicalGroundedness("아키텍처는 4계층 [1]", ev)
	extrinsic := LexicalGroundedness("금성의 대기압은 매우 높다", ev)
	if !(covered > extrinsic) {
		t.Fatalf("근거 내 답이 근거 밖 답보다 커버리지가 높아야 함: %f vs %f", covered, extrinsic)
	}
}
