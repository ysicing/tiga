// Copyright (c) 2023 ysicing(ysicing.me, ysicing@ysicing.cloud) All rights reserved.
// Use of this source code is covered by the following dual licenses:
// (1) Y PUBLIC LICENSE 1.0 (YPL 1.0)
// (2) Affero General Public License 3.0 (AGPL 3.0)
// License that can be found in the LICENSE file.

package main

import (
	"github.com/ysicing/tiga/cmd"
	"github.com/ysicing/tiga/cmd/boot"
)

func main() {
	if err := boot.OnBoot(); err != nil {
		panic(err)
	}
	cmd.Execute()
}
