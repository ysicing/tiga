package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/websocket"
	"k8s.io/apimachinery/pkg/types"

	"github.com/ysicing/tiga/internal/config"
	"github.com/ysicing/tiga/internal/models"
	k8sservice "github.com/ysicing/tiga/internal/services/k8s"
	"github.com/ysicing/tiga/pkg/cluster"
	"github.com/ysicing/tiga/pkg/common"
	"github.com/ysicing/tiga/pkg/kube"
	"github.com/ysicing/tiga/pkg/rbac"
	"github.com/ysicing/tiga/pkg/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NodeTerminalHandler struct {
	recordingService *k8sservice.TerminalRecordingService
}

func NewNodeTerminalHandler(recordingService *k8sservice.TerminalRecordingService) *NodeTerminalHandler {
	return &NodeTerminalHandler{
		recordingService: recordingService,
	}
}

// HandleNodeTerminalWebSocket handles WebSocket connections for node terminal access
func (h *NodeTerminalHandler) HandleNodeTerminalWebSocket(c *gin.Context) {
	cs := c.MustGet("cluster").(*cluster.ClientSet)

	nodeName := c.Param("nodeName")
	if nodeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Node name is required"})
		return
	}

	user := c.MustGet("user").(models.User)

	// Get node terminal image from config or use default
	nodeTerminalImage := "busybox:latest" // default
	if cfg, exists := c.Get("config"); exists {
		if appCfg, ok := cfg.(*config.Config); ok {
			nodeTerminalImage = appCfg.Kubernetes.NodeTerminalImage
		}
	}

	websocket.Handler(func(conn *websocket.Conn) {
		var recordingID uuid.UUID
		var recordingStopFunc func()

		defer func() {
			_ = conn.Close()
			// Stop recording if it was started
			if recordingStopFunc != nil {
				recordingStopFunc()
			}
		}()

		if !rbac.CanAccess(user, "nodes", "exec", cs.Name, "") {
			h.sendErrorMessage(conn, rbac.NoAccess(user.Key(), string(common.VerbExec), "nodes", "", cs.Name))
			return
		}

		node, err := cs.K8sClient.ClientSet.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
		if err != nil {
			log.Printf("Failed to get node %s: %v", nodeName, err)
			h.sendErrorMessage(conn, fmt.Sprintf("Failed to get node %s: %v", nodeName, err))
			return
		}

		if node == nil {
			log.Printf("Node %s not found", nodeName)
			h.sendErrorMessage(conn, fmt.Sprintf("Node %s not found", nodeName))
			return
		}

		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		nodeAgentName, err := h.createNodeAgent(ctx, cs, nodeName, nodeTerminalImage)
		if err != nil {
			log.Printf("Failed to create node agent pod: %v", err)
			h.sendErrorMessage(conn, fmt.Sprintf("Failed to create node agent pod: %v", err))
			return
		}

		// Ensure cleanup of the node agent pod
		defer func() {
			logrus.Infof("Cleaning up node agent pod %s", nodeAgentName)
			if err := h.cleanupNodeAgentPod(cs, nodeAgentName); err != nil {
				log.Printf("Failed to cleanup node agent pod %s: %v", nodeAgentName, err)
			}
		}()

		if err := h.waitForPodReady(ctx, cs, conn, nodeAgentName); err != nil {
			log.Printf("Failed to wait for pod ready: %v", err)
			h.sendErrorMessage(conn, fmt.Sprintf("Failed to wait for pod ready: %v", err))
			return
		}

		// Start terminal recording (T019 integration)
		k8sSession, _, recordingModel, err := h.recordingService.StartNodeTerminalRecording(
			ctx,
			user.ID,
			cs.Name,
			nodeName,
			80, // default width
			24, // default height
		)
		if err != nil {
			log.Printf("Failed to start terminal recording: %v", err)
			h.sendErrorMessage(conn, fmt.Sprintf("Failed to start terminal recording: %v", err))
			return
		}

		recordingID = recordingModel.ID
		recordingStopFunc = func() {
			if err := h.recordingService.StopRecording(ctx, recordingID, "session_ended"); err != nil {
				logrus.Errorf("Failed to stop recording: %v", err)
			}
		}

		// Create terminal session with recording support
		session := kube.NewTerminalSessionWithRecording(cs.K8sClient, conn, "kube-system", nodeAgentName, common.NodeTerminalPodName, k8sSession)
		if err := session.Start(ctx, "attach"); err != nil {
			logrus.Errorf("Terminal session error: %v", err)
		}
	}).ServeHTTP(c.Writer, c.Request)
}

func (h *NodeTerminalHandler) createNodeAgent(ctx context.Context, cs *cluster.ClientSet, nodeName string, nodeTerminalImage string) (string, error) {
	truncateNodeName := nodeName
	if len(nodeName)+len(common.NodeTerminalPodName)+5 > 63 {
		maxLength := 63 - len(common.NodeTerminalPodName) - 5
		truncateNodeName = nodeName[:maxLength]
	}
	podName := fmt.Sprintf("%s-%s-%s", common.NodeTerminalPodName, truncateNodeName, utils.RandomString(5))

	// Define the tiga node agent pod spec
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: "kube-system",
			Labels: map[string]string{
				"app": podName,
			},
		},
		Spec: corev1.PodSpec{
			NodeName:      nodeName,
			HostNetwork:   true,
			HostPID:       true,
			HostIPC:       true,
			RestartPolicy: corev1.RestartPolicyNever,
			Tolerations: []corev1.Toleration{
				{
					Operator: corev1.TolerationOpExists,
				},
			},
			Containers: []corev1.Container{
				{
					Name:  common.NodeTerminalPodName,
					Image: nodeTerminalImage,
					Command: []string{
						"nsenter",
						"--target", "1",
						"--mount", "--uts", "--ipc", "--net", "--pid",
						"--", "bash", "-c", "cd ~ && exec bash -l",
					},
					Stdin:     true,
					StdinOnce: true,
					TTY:       true,
					SecurityContext: &corev1.SecurityContext{
						Privileged: &[]bool{true}[0],
					},
				},
			},
		},
	}

	object := &corev1.Pod{}
	namespacedName := types.NamespacedName{Name: podName, Namespace: "kube-system"}
	if err := cs.K8sClient.Get(ctx, namespacedName, object); err == nil {
		if utils.IsPodErrorOrSuccess(object) {
			if err := cs.K8sClient.Delete(ctx, object); err != nil {
				return "", fmt.Errorf("failed to delete existing tiga node agent pod: %w", err)
			}
		} else {
			return podName, nil
		}
	}

	// Create the pod
	err := cs.K8sClient.Create(ctx, pod)
	if err != nil {
		return "", fmt.Errorf("failed to create tiga node agent pod: %w", err)
	}

	return podName, nil
}

// waitForPodReady waits for the tiga node agent pod to be ready
func (h *NodeTerminalHandler) waitForPodReady(ctx context.Context, cs *cluster.ClientSet, conn *websocket.Conn, podName string) error {
	timeout := time.After(60 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	h.sendMessage(conn, "info", fmt.Sprintf("waiting for pod %s to be ready", podName))

	var pod *corev1.Pod
	var err error
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timeout:
			h.sendMessage(conn, "info", "")
			h.sendErrorMessage(conn, utils.GetPodErrorMessage(pod))
			return fmt.Errorf("timeout waiting for pod %s to be ready", podName)
		case <-ticker.C:
			pod, err = cs.K8sClient.ClientSet.CoreV1().Pods("kube-system").Get(
				context.TODO(),
				podName,
				metav1.GetOptions{},
			)
			if err != nil {
				continue
			}
			h.sendMessage(conn, "stdout", ".")
			if utils.IsPodReady(pod) {
				h.sendMessage(conn, "info", "ready!")
				return nil
			}
		}
	}
}

func (h *NodeTerminalHandler) cleanupNodeAgentPod(cs *cluster.ClientSet, podName string) error {
	return cs.K8sClient.ClientSet.CoreV1().Pods("kube-system").Delete(
		context.TODO(),
		podName,
		metav1.DeleteOptions{},
	)
}

// sendErrorMessage sends an error message through WebSocket
func (h *NodeTerminalHandler) sendErrorMessage(conn *websocket.Conn, message string) {
	msg := map[string]interface{}{
		"type": "error",
		"data": message,
	}
	if err := websocket.JSON.Send(conn, msg); err != nil {
		log.Printf("Failed to send error message: %v", err)
	}
}

// sendErrorMessage sends an error message through WebSocket
func (h *NodeTerminalHandler) sendMessage(conn *websocket.Conn, msgType, message string) {
	msg := map[string]interface{}{
		"type": msgType,
		"data": message,
	}
	if err := websocket.JSON.Send(conn, msg); err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}
