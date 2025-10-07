package internal

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/ysicing/tiga/pkg/cluster"
	"github.com/ysicing/tiga/pkg/rbac"
)

var (
	tigaUsername = os.Getenv("tiga_USERNAME")
	tigaPassword = os.Getenv("tiga_PASSWORD")
)

func loadUser() error {
	if tigaUsername != "" && tigaPassword != "" {
		uc, err := model.CountUsers()
		if err == nil && uc == 0 {
			logrus.Infof("Creating super user %s from environment variables", tigaUsername)
			u := &model.User{
				Username: tigaUsername,
				Password: tigaPassword,
			}
			err := model.AddSuperUser(u)
			if err == nil {
				rbac.SyncNow <- struct{}{}
			} else {
				return err
			}
		}
	}

	return nil
}

func loadClusters() error {
	cc, err := model.CountClusters()
	if err != nil || cc > 0 {
		return err
	}
	kubeconfigpath := ""
	if home := homedir.HomeDir(); home != "" {
		kubeconfigpath = filepath.Join(home, ".kube", "config")
	}

	if envKubeconfig := os.Getenv("KUBECONFIG"); envKubeconfig != "" {
		kubeconfigpath = envKubeconfig
	}

	config, _ := os.ReadFile(kubeconfigpath)

	if len(config) == 0 {
		return nil
	}
	kubeconfig, err := clientcmd.Load(config)
	if err != nil {
		return err
	}

	logrus.Infof("Importing clusters from kubeconfig: %s", kubeconfigpath)
	cluster.ImportClustersFromKubeconfig(kubeconfig)
	return nil
}

// LoadConfigFromEnv loads configuration from environment variables.
func LoadConfigFromEnv() {
	if err := loadUser(); err != nil {
		logrus.Warnf("Failed to migrate env to db user: %v", err)
	}

	if err := loadClusters(); err != nil {
		logrus.Warnf("Failed to migrate env to db cluster: %v", err)
	}
}
