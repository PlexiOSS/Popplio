// Copyright (C) 2026 NodeByte LTD

package apps

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"popplio/types"
)

const (
	KindShort  = "short"
	KindLong   = "long"
	KindChoice = "choice"
	KindMulti  = "multi"
	KindYesNo  = "yesno"
	KindNumber = "number"
	KindURL    = "url"
	KindRating = "rating"

	shortMax   = 4096
	longMin    = 50
	longMax    = 10000
	maxOptions = 20
)

var Kinds = []string{KindShort, KindLong, KindChoice, KindMulti, KindYesNo, KindNumber, KindURL, KindRating}

var yesNo = []string{"Yes", "No"}

func Kind(q types.Question) string {
	if q.Type != "" {
		return q.Type
	}

	if q.Short {
		return KindShort
	}

	return KindLong
}

func Choices(q types.Question) []string {
	switch Kind(q) {
	case KindYesNo:
		return yesNo
	case KindRating:
		return []string{"1", "2", "3", "4", "5"}
	}

	return q.Options
}

func lengthLimits(q types.Question) (int, int) {
	lo, hi := 1, shortMax

	if Kind(q) == KindLong {
		lo, hi = longMin, longMax
	}

	if q.MinLength > 0 {
		lo = q.MinLength
	}

	if q.MaxLength > 0 {
		hi = q.MaxLength
	}

	return lo, hi
}

func SplitMulti(answer string) []string {
	var out []string

	for _, part := range strings.Split(answer, "\n") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}

	return out
}

func Visible(q types.Question, answers map[string]string) bool {
	if q.ShowIf == nil {
		return true
	}

	given := SplitMulti(answers[q.ShowIf.QuestionID])

	for _, want := range q.ShowIf.Values {
		if slices.Contains(given, want) {
			return true
		}
	}

	return false
}

func CheckAnswer(q types.Question, answer string) error {
	switch Kind(q) {
	case KindShort, KindLong:
		lo, hi := lengthLimits(q)
		n := utf8.RuneCountInString(answer)

		if n < lo {
			return fmt.Errorf("needs at least %d characters", lo)
		}

		if n > hi {
			return fmt.Errorf("can be at most %d characters", hi)
		}
	case KindChoice, KindYesNo, KindRating:
		if !slices.Contains(Choices(q), answer) {
			return errors.New("pick one of the options")
		}
	case KindMulti:
		picked := SplitMulti(answer)

		if len(picked) == 0 {
			return errors.New("pick at least one option")
		}

		seen := map[string]bool{}

		for _, p := range picked {
			if !slices.Contains(q.Options, p) || seen[p] {
				return errors.New("pick from the options, each once")
			}

			seen[p] = true
		}
	case KindNumber:
		if len(answer) > 32 {
			return errors.New("is too long for a number")
		}

		if _, err := strconv.ParseFloat(answer, 64); err != nil {
			return errors.New("needs to be a number")
		}
	case KindURL:
		u, err := url.Parse(answer)

		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || len(answer) > 500 {
			return errors.New("needs to be a link starting with https://")
		}
	default:
		return errors.New("has an unknown question type")
	}

	return nil
}

func CheckDefinition(questions []types.Question) string {
	index := map[string]int{}

	for i, q := range questions {
		label := fmt.Sprintf("Question %d", i+1)
		kind := Kind(q)

		if !slices.Contains(Kinds, kind) {
			return label + " has an unknown type"
		}

		switch kind {
		case KindChoice, KindMulti:
			if len(q.Options) < 2 || len(q.Options) > maxOptions {
				return fmt.Sprintf("%s needs between 2 and %d options", label, maxOptions)
			}

			seen := map[string]bool{}

			for _, o := range q.Options {
				if o == "" || len(o) > 100 || strings.Contains(o, "\n") || seen[o] {
					return label + "'s options must be different, one line and up to 100 characters each"
				}

				seen[o] = true
			}
		default:
			if len(q.Options) > 0 {
				return label + " can only have options if it's single or multiple choice"
			}
		}

		if q.MinLength < 0 || q.MaxLength < 0 || (q.MaxLength > 0 && q.MinLength > q.MaxLength) {
			return label + "'s length limits don't make sense"
		}

		if (kind == KindShort && q.MaxLength > shortMax) || (kind == KindLong && q.MaxLength > longMax) {
			return label + "'s maximum length is too high"
		}

		if len(q.Flag) > 0 {
			choices := Choices(q)

			if len(choices) == 0 {
				return label + " can only flag answers if it's a choice, yes or no, or rating question"
			}

			for _, v := range q.Flag {
				if !slices.Contains(choices, v) {
					return fmt.Sprintf("%s flags an answer (%q) it doesn't offer", label, v)
				}
			}
		}

		if q.ShowIf != nil {
			j, ok := index[q.ShowIf.QuestionID]

			if !ok {
				return label + " can only depend on a question that comes before it"
			}

			choices := Choices(questions[j])

			if len(choices) == 0 {
				return label + " can only depend on a choice, yes or no, or rating question"
			}

			if len(q.ShowIf.Values) == 0 {
				return label + " needs at least one answer to show for"
			}

			for _, v := range q.ShowIf.Values {
				if !slices.Contains(choices, v) {
					return fmt.Sprintf("%s depends on an answer (%q) that question %d doesn't offer", label, v, j+1)
				}
			}
		}

		index[q.ID] = i
	}

	return ""
}
