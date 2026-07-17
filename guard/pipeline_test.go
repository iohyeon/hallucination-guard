package guard

import (
	"context"
	"testing"

	"github.com/iohyeon/hallucination-guard/backend"
)

func loopersEvidence() []Evidence {
	return []Evidence{
		{ID: 1, Text: "Loopers 아키텍처는 4계층으로 구성된다"},
		{ID: 2, Text: "모든 Controller 는 Facade 만 호출한다"},
	}
}

// 근거가 충분하고 mock 이 인용된 grounded 답을 내면 answer 로 통과한다.
func TestPipeline_GroundedAnswerPasses(t *testing.T) {
	p := New(backend.NewMock(), Default())
	r, err := p.Run(context.Background(), "Loopers 아키텍처 계층", loopersEvidence())
	if err != nil {
		t.Fatal(err)
	}
	if r.Decision != Answer {
		t.Fatalf("통과해야 하는데: decision=%s reasons=%v", r.Decision, r.Reasons)
	}
}

// 관련 근거가 없으면 생성 전에 기권한다.
func TestPipeline_NoEvidenceAbstains(t *testing.T) {
	p := New(backend.NewMock(), Default())
	ev := []Evidence{{ID: 1, Text: "고양이는 포유류다"}}
	r, err := p.Run(context.Background(), "quantum computing", ev)
	if err != nil {
		t.Fatal(err)
	}
	if r.Decision != Abstain {
		t.Fatalf("근거 없음 → 기권이어야 함: %s", r.Decision)
	}
	if r.Answer != "문서에 근거 없음" {
		t.Fatalf("기권 응답 문구 불일치: %q", r.Answer)
	}
}

// 허위 인용 번호는 규칙 층에서 반려된다.
func TestPipeline_FabricatedCitationAbstains(t *testing.T) {
	m := &backend.Mock{Answer: "지구는 평평하다 [9].", JudgeGrounded: true}
	p := New(m, Default())
	r, err := p.Run(context.Background(), "Loopers 아키텍처 계층", loopersEvidence())
	if err != nil {
		t.Fatal(err)
	}
	if r.Decision != Abstain {
		t.Fatalf("허위 인용 → 기권이어야 함: %s (%v)", r.Decision, r.Reasons)
	}
}

// judge 가 미충실로 판정하면 반려된다.
func TestPipeline_JudgeUngroundedAbstains(t *testing.T) {
	m := &backend.Mock{Answer: "아키텍처는 4계층이다 [1].", JudgeGrounded: false}
	p := New(m, Default())
	r, err := p.Run(context.Background(), "Loopers 아키텍처 계층", loopersEvidence())
	if err != nil {
		t.Fatal(err)
	}
	if r.Decision != Abstain {
		t.Fatalf("judge 미충실 → 기권이어야 함: %s (%v)", r.Decision, r.Reasons)
	}
	if r.Verdict == nil || r.Verdict.Grounded {
		t.Fatalf("verdict 가 미충실로 기록돼야 함: %+v", r.Verdict)
	}
}
