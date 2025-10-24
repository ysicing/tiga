package collector

import (
	"context"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	dockerclient "github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

// DockerInfo Docker信息
type DockerInfo struct {
	Installed         bool
	Version           string
	APIVersion        string
	OS                string
	Arch              string
	KernelVersion     string
	StorageDriver     string
	Containers        int32
	ContainersRunning int32
	ContainersPaused  int32
	ContainersStopped int32
	Images            int32
	MemTotal          uint64
	NCPU              int32
}

// CollectDockerInfo 收集Docker信息
// 如果Docker未安装或无法连接,返回installed=false
func (c *Collector) CollectDockerInfo() *DockerInfo {
	info := &DockerInfo{
		Installed: false,
	}

	// 尝试创建Docker客户端
	// 优先使用环境变量，如果失败则尝试默认socket
	var cli *dockerclient.Client
	var err error

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 首先尝试使用环境变量
	cli, err = dockerclient.NewClientWithOpts(
		dockerclient.FromEnv,
		dockerclient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		logrus.Debugf("Docker client with env failed: %v", err)
	} else {
		// 尝试Ping Docker daemon
		_, err = cli.Ping(ctx)
		if err != nil {
			logrus.Debugf("Docker daemon ping with env failed: %v", err)
			cli.Close()
			cli = nil
		}
	}

	// 如果环境变量方式失败，尝试使用默认socket
	if cli == nil {
		logrus.Debug("Trying default Docker socket...")
		cli, err = dockerclient.NewClientWithOpts(
			dockerclient.WithHost("unix:///var/run/docker.sock"),
			dockerclient.WithAPIVersionNegotiation(),
		)
		if err != nil {
			logrus.Debugf("Docker client creation failed (this is normal if Docker is not installed): %v", err)
			return info
		}

		// 尝试Ping Docker daemon
		_, err = cli.Ping(ctx)
		if err != nil {
			logrus.Debugf("Docker daemon ping failed: %v", err)
			cli.Close()
			return info
		}
	}
	defer cli.Close()

	// Docker已安装并运行，获取版本信息
	version, err := cli.ServerVersion(ctx)
	if err != nil {
		logrus.Warnf("Failed to get Docker version: %v", err)
		return info
	}

	// 获取Docker信息
	dockerInfo, err := cli.Info(ctx)
	if err != nil {
		logrus.Warnf("Failed to get Docker info: %v", err)
		// 仅有版本信息也算成功
		info.Installed = true
		info.Version = version.Version
		info.APIVersion = version.APIVersion
		info.OS = version.Os
		info.Arch = version.Arch
		info.KernelVersion = version.KernelVersion
		return info
	}

	// 填充完整信息
	info.Installed = true
	info.Version = version.Version
	info.APIVersion = version.APIVersion
	info.OS = version.Os
	info.Arch = version.Arch
	info.KernelVersion = version.KernelVersion
	info.StorageDriver = dockerInfo.Driver
	info.Containers = int32(dockerInfo.Containers)
	info.ContainersRunning = int32(dockerInfo.ContainersRunning)
	info.ContainersPaused = int32(dockerInfo.ContainersPaused)
	info.ContainersStopped = int32(dockerInfo.ContainersStopped)
	info.Images = int32(dockerInfo.Images)
	info.MemTotal = uint64(dockerInfo.MemTotal)
	info.NCPU = int32(dockerInfo.NCPU)

	logrus.WithFields(logrus.Fields{
		"version":    info.Version,
		"containers": info.Containers,
		"images":     info.Images,
	}).Debug("Docker info collected successfully")

	return info
}

// GetDockerStatus 获取Docker运行状态(用于定期更新)
func (c *Collector) GetDockerStatus() (running bool, containers int32, images int32) {
	cli, err := dockerclient.NewClientWithOpts(
		dockerclient.FromEnv,
		dockerclient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return false, 0, 0
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 检查Docker是否运行
	_, err = cli.Ping(ctx)
	if err != nil {
		return false, 0, 0
	}

	// 获取容器统计
	containerList, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err == nil {
		containers = int32(len(containerList))
	}

	// 获取镜像统计
	imageList, err := cli.ImageList(ctx, image.ListOptions{})
	if err == nil {
		images = int32(len(imageList))
	}

	return true, containers, images
}
