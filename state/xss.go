// Copyright (C) 2026 NodeByte LTD

package state

import (
	"reflect"
	"regexp"

	"github.com/go-playground/validator/v10"
)

var xssPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)<\s*script\b`),
	regexp.MustCompile(`(?i)javascript\s*:`),
	regexp.MustCompile(`(?i)vbscript\s*:`),
	regexp.MustCompile(`(?i)data\s*:\s*text/html`),
	regexp.MustCompile(`(?i)on[a-z]+\s*=\s*["']?`),
	regexp.MustCompile(`(?i)<\s*svg\b`),
	regexp.MustCompile(`(?i)<\s*object\b`),
	regexp.MustCompile(`(?i)<\s*embed\b`),
	regexp.MustCompile(`(?i)<\s*meta\b`),
	regexp.MustCompile(`(?i)expression\s*\(`),
}

func ContainsSuspiciousMarkup(text string) bool {
	for _, p := range xssPatterns {
		if p.MatchString(text) {
			return true
		}
	}
	return false
}

func noXSS(fl validator.FieldLevel) bool {
	switch fl.Field().Kind() {
	case reflect.String:
		return !ContainsSuspiciousMarkup(fl.Field().String())
	default:
		return true
	}
}
