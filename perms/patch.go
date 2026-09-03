// Copyright (C) 2026 NodeByte LTD

package perms

import (
	"errors"
	"fmt"
)

var ErrCannotManage = errors.New("insufficient permission to manage this permission")

func CheckPatch(manager, current, next Set) error {
	for _, p := range current.Diff(next) {
		if !manager.Has(p) {
			return fmt.Errorf("%w: %s", ErrCannotManage, p)
		}
	}

	return nil
}

func CanGrant(manager Set, p Perm) error {
	if !manager.Has(p) {
		return fmt.Errorf("%w: %s", ErrCannotManage, p)
	}

	return nil
}

func Unmanageable(manager, current, next Set) []Perm {
	var out []Perm

	for _, p := range current.Diff(next) {
		if !manager.Has(p) {
			out = append(out, p)
		}
	}

	return out
}
