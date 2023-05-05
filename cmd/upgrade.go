// Copyright (c) 2023 ysicing(ysicing.me, ysicing@ysicing.cloud) All rights reserved.
// Use of this source code is covered by the following dual licenses:
// (1) Y PUBLIC LICENSE 1.0 (YPL 1.0)
// (2) Affero General Public License 3.0 (AGPL 3.0)
// License that can be found in the LICENSE file.

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/ysicing/tiga/cmd/upgrade"
	"github.com/ysicing/tiga/pkg/factory"
)

func newCmdUpgrade(f factory.Factory) *cobra.Command {
	up := &cobra.Command{
		Use:     "upgrade",
		Short:   "Upgrade tiga to the latest version",
		Aliases: []string{"ug", "ugc"},
		Run: func(cmd *cobra.Command, args []string) {
			upgrade.NewUpgradeTiga(f)
		},
	}
	return up
}
