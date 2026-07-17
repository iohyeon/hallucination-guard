package guard

// MaxSimilarity 는 질의와 근거 조각들 사이 최고 어휘 유사도와 그 근거 ID 를
// 돌려준다. 예방 층의 게이트: 어떤 근거도 충분히 관련되지 않으면 사실상
// 그라운딩이 없는 것이므로, 생성 전에 반려(abstain)해 모델이 빈 근거 위에서
// 지어내는 것을 막는다.
func MaxSimilarity(query string, evidence []Evidence) (float64, int) {
	q := termFreq(tokenize(query))
	best, bestID := 0.0, -1
	for _, e := range evidence {
		s := cosine(q, termFreq(tokenize(e.Text)))
		if s > best {
			best, bestID = s, e.ID
		}
	}
	return best, bestID
}

// LexicalGroundedness 는 답의 토큰이 근거에 얼마나 덮이는지를 [0,1] 로 잰다.
// 값이 낮으면 답이 근거에 없는 내용을 끌어들인 것(extrinsic 확장)이다.
// LLM 없이 도는 검출 신호로, 비싼 judge 앞에 두는 싼 층이다.
func LexicalGroundedness(answer string, evidence []Evidence) float64 {
	ev := map[string]bool{}
	for _, e := range evidence {
		for _, t := range tokenize(e.Text) {
			ev[t] = true
		}
	}
	toks := tokenize(answer)
	if len(toks) == 0 {
		return 0
	}
	covered := 0
	for _, t := range toks {
		if ev[t] {
			covered++
		}
	}
	return float64(covered) / float64(len(toks))
}
