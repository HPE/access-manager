/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package common

import "regexp"

const (
	StandardPrefix = "am://"
	UserPrefix     = StandardPrefix + "user"
	WorkloadPrefix = StandardPrefix + "workload"
	DataPrefix     = StandardPrefix + "data"
	RolePrefix     = StandardPrefix + "role"
	KeyPrefix      = StandardPrefix + "key"

	AnonymousUser     = UserPrefix + "/***/***"
	AnonymousWorkload = WorkloadPrefix + "/***/***"
	AnonymousDataset  = DataPrefix + "/***/***"
	RedactedRole      = "## Redacted role ##"
)

var (
	StandardPattern = regexp.MustCompile(`^` + StandardPrefix + `([a-zA-Z0-9-]+(/[a-zA-Z0-9-]*)*/?)?$`)
)
