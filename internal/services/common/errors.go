/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package common

type VersionMismatchError struct {
	Msg string
}

func (e VersionMismatchError) Error() string {
	return "version mismatch: " + e.Msg
}
