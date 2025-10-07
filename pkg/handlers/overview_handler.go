package handlers

import (
	"context"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/internal/repository"
	"github.com/ysicing/tiga/pkg/cluster"
	"github.com/ysicing/tiga/pkg/common"

	v1 "k8s.io/api/core/v1"
)

type OverviewData struct {
	TotalNodes      int                   `json:"totalNodes"`
	ReadyNodes      int                   `json:"readyNodes"`
	TotalPods       int                   `json:"totalPods"`
	RunningPods     int                   `json:"runningPods"`
	TotalNamespaces int                   `json:"totalNamespaces"`
	TotalServices   int                   `json:"totalServices"`
	PromEnabled     bool                  `json:"prometheusEnabled"`
	Resource        common.ResourceMetric `json:"resource"`
}

func GetOverview(c *gin.Context) {
	ctx := c.Request.Context()

	cs := c.MustGet("cluster").(*cluster.ClientSet)
	// Simplified: removed RBAC check, using is_admin flag if needed

	// Get nodes
	nodes := &v1.NodeList{}
	if err := cs.K8sClient.List(ctx, nodes, &client.ListOptions{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	readyNodes := 0
	var cpuAllocatable, memAllocatable resource.Quantity
	var cpuRequested, memRequested resource.Quantity
	var cpuLimited, memLimited resource.Quantity
	for _, node := range nodes.Items {
		cpuAllocatable.Add(*node.Status.Allocatable.Cpu())
		memAllocatable.Add(*node.Status.Allocatable.Memory())
		for _, condition := range node.Status.Conditions {
			if condition.Type == v1.NodeReady && condition.Status == v1.ConditionTrue {
				readyNodes++
				break
			}
		}
	}

	// Get pods
	pods := &v1.PodList{}
	if err := cs.K8sClient.List(ctx, pods, &client.ListOptions{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	runningPods := 0
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			cpuRequested.Add(*container.Resources.Requests.Cpu())
			memRequested.Add(*container.Resources.Requests.Memory())

			if container.Resources.Limits != nil {
				if cpuLimit := container.Resources.Limits.Cpu(); cpuLimit != nil {
					cpuLimited.Add(*cpuLimit)
				}
				if memLimit := container.Resources.Limits.Memory(); memLimit != nil {
					memLimited.Add(*memLimit)
				}
			}
		}
		if pod.Status.Phase == v1.PodRunning || pod.Status.Phase == v1.PodSucceeded {
			runningPods++
		}
	}

	// Get namespaces
	namespaces := &v1.NamespaceList{}
	if err := cs.K8sClient.List(ctx, namespaces, &client.ListOptions{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get services
	services := &v1.ServiceList{}
	if err := cs.K8sClient.List(ctx, services, &client.ListOptions{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	overview := OverviewData{
		TotalNodes:      len(nodes.Items),
		ReadyNodes:      readyNodes,
		TotalPods:       len(pods.Items),
		RunningPods:     runningPods,
		TotalNamespaces: len(namespaces.Items),
		TotalServices:   len(services.Items),
		PromEnabled:     cs.PromClient != nil,
		Resource: common.ResourceMetric{
			CPU: common.Resource{
				Allocatable: cpuAllocatable.MilliValue(),
				Requested:   cpuRequested.MilliValue(),
				Limited:     cpuLimited.MilliValue(),
			},
			Mem: common.Resource{
				Allocatable: memAllocatable.MilliValue(),
				Requested:   memRequested.MilliValue(),
				Limited:     memLimited.MilliValue(),
			},
		},
	}

	c.JSON(http.StatusOK, overview)
}

// var (
// 	initialized bool
// )

// AppConfig 配置结构 (与 install service 保持一致)
type AppConfig struct {
	Server struct {
		InstallLock bool `yaml:"install_lock"`
	} `yaml:"server"`
}

// NewInitCheckHandler 创建 InitCheck 处理器
func NewInitCheckHandler(configPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 检查配置文件中的 install_lock
		data, err := os.ReadFile(configPath)
		if err != nil {
			// 配置文件不存在或读取失败 = 未安装
			c.JSON(http.StatusOK, gin.H{
				"initialized": false,
				"step":        0,
			})
			return
		}

		var config AppConfig
		if err := yaml.Unmarshal(data, &config); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"initialized": false,
				"step":        0,
			})
			return
		}

		if config.Server.InstallLock {
			// ✅ install_lock = true → 已安装
			c.JSON(http.StatusOK, gin.H{
				"initialized": true,
				"step":        2,
			})
			return
		}

		// 2. 如果 install_lock = false，检查安装进度
		step := 0
		userRepo := repository.NewUserRepository(models.DB)
		uc, _ := userRepo.Count(context.Background())
		if uc > 0 || common.AnonymousUserEnabled {
			step = 1 // 已创建用户，进入步骤2
		}

		c.JSON(http.StatusOK, gin.H{
			"initialized": false,
			"step":        step,
		})
	}
}
