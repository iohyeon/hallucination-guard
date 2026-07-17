# hallucination-guard

LLM 할루시네이션을 **시스템으로** 막는 다층 방어 파이프라인. Go로 구현했다.

프롬프트 한 줄("거짓말하지 마")은 생성 확률을 약간 기울일 뿐 보장하지 못한다.
이 라이브러리는 생성물을 신뢰하지 않고, 여러 층이 실측(근거)에 대조해 통과·검토·반려로 게이트한다.

```
질의 + 근거
   │
   ▼  [예방]  그라운딩 게이트  ── 관련 근거 없으면 생성 전에 기권
   ▼          생성            ── 근거에 그라운딩 + 인용 강제
   ▼  [검출]  인용 규칙 검증   ── (싼 규칙, LLM 없음) 허위/누락 인용 차단
   ▼          어휘 groundedness ── (싼 신호) 근거 밖 확장 감지
   ▼          시맨틱 엔트로피   ── (선택, N표본) 불확실성 신호
   ▼          LLM-as-judge     ── (비쌈, 선택) 의미 수준 미충실 판정
   ▼  [게이트] answer / review / abstain
```

## 왜 만들었나

백엔드 개발자로서 "LLM 서빙을 안정적인 시스템으로 감싼다"를 코드로 익히려고 만들었다.
개념 학습 노트(예방·검출·완화·관측의 다층 방어)를 그대로 실행 가능한 Go 패키지로 옮겼다.

## 설계: 개념 → 코드 매핑

| 방어 개념 | 성격 | 코드 |
|---|---|---|
| 그라운딩(근거 주입, 지식 경계 축소) | 예방 | `guard.MaxSimilarity`, 파이프라인 게이트 |
| 인용 강제 + 규칙 검증 | 검출(1층) | `guard.CheckCitations` |
| 어휘 groundedness | 검출(싼 신호) | `guard.LexicalGroundedness` |
| 시맨틱 엔트로피(불확실성) | 검출(선택) | `guard.SemanticEntropy` |
| LLM-as-judge(충실성 판정) | 검출(2층) | `guard.LLMJudge` |
| 기권/검토 게이트 | 완화 | `guard.Pipeline.Run` → `Decision` |

싼 층을 앞에 두고 명백한 케이스를 먼저 쳐내, 비싼 LLM judge 호출 수를 줄인다.

## 백엔드: pluggable

가드 로직은 `backend.LLM` 인터페이스에만 의존한다(의존성 역전).

- `backend.Mock`: 오프라인·결정적. **API 키 없이** 파이프라인과 테스트를 돌린다.
- `backend.Claude`: 실제 Anthropic API. 키를 `ANTHROPIC_API_KEY` 환경변수에서 읽는다.

키는 코드에 하드코딩하지 않는다. `.env` 는 `.gitignore` 되어 커밋되지 않으며,
레포를 포크한 개발자는 **각자 자기 키를 넣고 자기 비용으로** 실행한다.

## 실행

전제: Go 1.23+ 설치 (`brew install go`).

```bash
git clone <this-repo> && cd hallucination-guard
go mod tidy          # anthropic-sdk-go 의존성 해석

# 1) 키 없이 오프라인 데모 (mock 백엔드)
go run ./cmd/halluguard --mock

# 2) 실제 Claude 로 데모 (본인 키)
cp .env.example .env         # .env 에 ANTHROPIC_API_KEY 채우기
export $(grep -v '^#' .env | xargs)
go run ./cmd/halluguard

# self-consistency(시맨틱 엔트로피) 활성화: 표본 4개
go run ./cmd/halluguard --mock --samples 4
```

### 테스트

규칙 기반 층은 LLM 없이 결정적으로 검증된다. 키·네트워크 없이 돈다.

```bash
go test ./...
```

### 라이브러리로 사용

```go
p := guard.New(backend.NewMock(), guard.Default())
r, _ := p.Run(ctx, "질문", []guard.Evidence{{ID: 1, Text: "근거..."}})
fmt.Println(r.Decision) // answer / review / abstain
```

## 출력 예시 (mock)

```
Q: 태양계에서 가장 큰 행성은?
A: 문서에 근거 없음
→ decision: abstain
   signal[grounding] max_similarity = 0.000  최고 유사 근거 [-1]
   reason: 근거 부족: 관련 근거를 찾지 못해 생성을 건너뜀
```

관련 근거가 없으니 지어낸 답 대신 기권한다.

## 정직한 한계 (학습용 근사)

이 레포는 개념을 코드로 보여주려는 것이라, 일부를 의도적으로 근사했다.

- **임베딩 없음**: Anthropic API 는 임베딩 엔드포인트를 제공하지 않는다. 그라운딩
  관련도를 어휘 코사인으로 근사했다. 실무는 임베딩 + ANN 검색(pgvector 등)으로 대체한다.
- **시맨틱 엔트로피의 어휘 근사**: 답을 정규화된 문자열 단위로 군집한다. 정식
  구현은 의미 동치를 함의 판정 모델로 군집해야 한다(패러프레이즈 과대평가 회피).
- **judge 자기 편향**: 생성과 판정을 같은 백엔드로 돌린다. 실무는 판정용 모델을
  분리하고 주장 분해·근거 스팬 요구·다수결로 신뢰도를 올린다.
- **구조화 출력**: judge JSON 을 프롬프트로 유도하고 관대하게 파싱한다. 실무는
  `output_config` 구조화 출력으로 스키마를 강제하는 편이 견고하다.

## 라이선스

MIT. 자세한 건 `LICENSE`.
