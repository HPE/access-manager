/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package main

import (
	"github.com/hpe/access-manager/cmd/am/cmd"
)

/*
This implements a command line management interface roughly in the style of kubectl but
for the Access Manager
*/
func main() {
	cmd.Init()
	cmd.Execute()
}
