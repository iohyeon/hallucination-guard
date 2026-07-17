package backend

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// Claude 는 실제 Anthropic API 를 호출한다. 키는 ANTHROPIC_API_KEY 환경변수에서
// 읽고(코드에 하드코딩하지 않는다), 모델은 ANTHROPIC_MODEL 에서 읽되 없으면
// claude-opus-4-8 을 쓴다. 레포를 포크한 개발자는 각자 자기 키로 자기 비용을
// 지불하며 실행한다.
type Claude struct {
	client anthropic.Client
	model  anthropic.Model
	maxTok int64
}

// NewClaude 는 클라이언트를 만든다. 키가 없으면 즉시 에러를 반환해 호출부가
// Mock 으로 폴백할 수 있게 한다.
func NewClaude() (*Claude, error) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY 미설정: 실제 백엔드를 쓰려면 키가 필요하다")
	}
	model := anthropic.ModelClaudeOpus4_8
	if m := os.Getenv("ANTHROPIC_MODEL"); m != "" {
		model = anthropic.Model(m)
	}
	return &Claude{
		client: anthropic.NewClient(), // ANTHROPIC_API_KEY 를 자동으로 읽는다
		model:  model,
		maxTok: 1024,
	}, nil
}

func (c *Claude) Name() string { return "claude(" + string(c.model) + ")" }

// Generate 는 Messages API 를 한 번 호출하고 텍스트 블록을 이어붙여 반환한다.
func (c *Claude) Generate(ctx context.Context, system, user string) (string, error) {
	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: c.maxTok,
		System:    []anthropic.TextBlockParam{{Text: system}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(user)),
		},
	})
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			sb.WriteString(t.Text)
		}
	}
	return sb.String(), nil
}
