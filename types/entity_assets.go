// Copyright (C) 2026 NodeByte LTD

package types

type PutEntityAssetVersion struct {
	Version string `json:"version" description:"Lowercase hex prefix (16 to 64 chars) of the uploaded file's SHA-256"`
}
