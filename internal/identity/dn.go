// Copyright (c) 2026 Deployment Theory.
// SPDX-License-Identifier: MIT

package identity

import "github.com/deploymenttheory/go-sdk-activedirectory/activedirectory/shared"

// ParentDN preserves wire spelling while locating the first unescaped RFC4514 separator after SDK validation.
func ParentDN(dn string) (string, bool) {
	if _, err := shared.IdentityByDN(dn); err != nil {
		return "", false
	}
	for index := 0; index < len(dn); index++ {
		if dn[index] == '\\' {
			index++
			continue
		}
		if dn[index] == ',' {
			return dn[index+1:], true
		}
	}
	return "", false
}
