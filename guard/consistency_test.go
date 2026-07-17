package guard

import (
	"context"
	"testing"

	"github.com/iohyeon/hallucination-guard/backend"
)

func TestSemanticEntropy_IdenticalSamplesAreConfident(t *testing.T) {
	m := &backend.Mock{Answer: "동일한 답 [1]"}
	ent, samples, err := SemanticEntropy(context.Background(), m, "sys", "user", 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 4 {
		t.Fatalf("표본 수 불일치: %d", len(samples))
	}
	if ent != 0 {
		t.Fatalf("동일 표본은 엔트로피 0(확신)이어야 함: %f", ent)
	}
}
