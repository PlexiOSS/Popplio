// Copyright (C) 2026 NodeByte LTD

package types

type ListTag struct {
	Tag     string `json:"tag" description:"The tag, in its most common capitalisation"`
	Bots    int64  `json:"bots" description:"How many listed bots use it"`
	Servers int64  `json:"servers" description:"How many public listed servers use it"`
}

type ListTags struct {
	Tags []ListTag `json:"tags" description:"Every tag in use, most used first"`
}
