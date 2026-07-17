// Command halluguard 는 다층 할루시네이션 가드 데모를 돌린다.
// ANTHROPIC_API_KEY 가 있으면 실제 Claude 로, 없으면(또는 --mock) 오프라인
// mock 백엔드로 동작한다.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/iohyeon/hallucination-guard/backend"
	"github.com/iohyeon/hallucination-guard/guard"
)

func main() {
	useMock := flag.Bool("mock", false, "오프라인 mock 백엔드를 강제 사용")
	samples := flag.Int("samples", 0, "self-consistency 표본 수(>=2 면 활성)")
	flag.Parse()

	llm := pickBackend(*useMock)
	fmt.Printf("backend = %s\n\n", llm.Name())

	cfg := guard.Default()
	cfg.ConsistencySamples = *samples
	p := guard.New(llm, cfg)

	cases := []struct {
		query    string
		evidence []guard.Evidence
	}{
		{
			query: "Loopers 아키텍처의 계층 구조는?",
			evidence: []guard.Evidence{
				{ID: 1, Text: "Loopers 아키텍처는 Interfaces, Application, Domain, Infrastructure 4계층으로 구성된다."},
				{ID: 2, Text: "모든 Controller 는 Facade 만 호출한다."},
			},
		},
		{
			query: "태양계에서 가장 큰 행성은?",
			evidence: []guard.Evidence{
				{ID: 1, Text: "Loopers 아키텍처는 4계층으로 구성된다."},
			},
		},
	}

	for _, c := range cases {
		r, err := p.Run(context.Background(), c.query, c.evidence)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		printResult(r)
	}
}

func pickBackend(forceMock bool) backend.LLM {
	if !forceMock {
		if c, err := backend.NewClaude(); err == nil {
			return c
		}
	}
	return backend.NewMock()
}

func printResult(r *guard.Result) {
	fmt.Printf("Q: %s\n", r.Query)
	fmt.Printf("A: %s\n", r.Answer)
	fmt.Printf("→ decision: %s\n", r.Decision)
	for _, s := range r.Signals {
		fmt.Printf("   signal[%s] %s = %.3f  %s\n", s.Layer, s.Name, s.Value, s.Note)
	}
	for _, is := range r.CitationIssues {
		fmt.Printf("   citation[%s] %s: %q\n", is.Kind, is.Detail, is.Sentence)
	}
	if r.Verdict != nil {
		fmt.Printf("   judge.grounded = %v (%s)\n", r.Verdict.Grounded, r.Verdict.Reason)
	}
	for _, why := range r.Reasons {
		fmt.Printf("   reason: %s\n", why)
	}
	fmt.Println()
}
