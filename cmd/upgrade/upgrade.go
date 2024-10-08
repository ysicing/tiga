// Copyright (c) 2023 ysicing(ysicing.me, ysicing@12306.work) All rights reserved.
// Use of this source code is covered by the following dual licenses:
// (1) Y PUBLIC LICENSE 1.0 (YPL 1.0)
// (2) Affero General Public License 3.0 (AGPL 3.0)
// License that can be found in the LICENSE file.

package upgrade

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	gv "github.com/Masterminds/semver/v3"
	"github.com/ysicing/tiga/cmd/version"
	"github.com/ysicing/tiga/common"
	"github.com/ysicing/tiga/pkg/factory"
	"github.com/ysicing/tiga/pkg/log"
	"github.com/ysicing/tiga/pkg/selfupdate"
)

type option struct {
	log log.Logger
}

func NewUpgradeTiga(f factory.Factory) {
	up := option{
		log: f.GetLog(),
	}
	up.DoTiGA()
}

func (up option) DoTiGA() {
	up.log.Info("fetch latest version from remote...")
	lastVersion, lastType, err := version.PreCheckLatestVersion(up.log)
	if err != nil {
		up.log.Errorf("fetch latest version err, reason: %v", err)
		return
	}
	if lastVersion == "" || lastVersion == common.Version || strings.Contains(common.Version, lastVersion) {
		up.log.Infof("current version %s is the latest version", common.Version)
		return
	}
	if lastVersion != "" && !strings.Contains(common.Version, lastVersion) {
		nowVersion := gv.MustParse(strings.TrimPrefix(common.Version, "v"))
		needUpgrade := nowVersion.LessThan(gv.MustParse(lastVersion))
		if !needUpgrade {
			up.log.Infof("current version %s is the newest version", common.Version)
			return
		}
	}
	cmdPath, err := os.Executable()
	if err != nil {
		up.log.Errorf("tiga executable err:%v", err)
		return
	}
	up.log.Infof("downloading version %s...", lastVersion)
	assetURL := fmt.Sprintf("https://gh.dev.438250.xyz/https://github.com/ysicing/tiga/releases/download/%s/tiga_%s_%s", lastVersion, runtime.GOOS, runtime.GOARCH)

	if lastType == "api" {
		// TODO 暂不支持
		panic("not support now from api")
	}
	err = selfupdate.UpdateTo(up.log, assetURL, cmdPath)
	if err != nil {
		up.log.Errorf("upgrade failed, reason: %v", err)
		return
	}
	up.log.Donef("updated tiga to version %s", lastVersion)
	up.log.Debugf("gen new version manifest")
	up.log.Infof("Release note: \n\t release %s ", lastVersion)
	up.log.Infof("Upgrade docs: \n\t https://github.com/ysicing/tiga/releases")
}
