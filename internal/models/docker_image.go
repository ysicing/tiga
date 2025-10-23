package models

import "time"

// Image represents a Docker image (non-persistent, from Agent)
type Image struct {
	ID           string            `json:"id"`           // Image ID (short format)
	LongID       string            `json:"long_id"`      // Full image ID
	RepoTags     []string          `json:"repo_tags"`    // Image tags (e.g., ["nginx:latest", "nginx:1.25"])
	RepoDigests  []string          `json:"repo_digests"` // Image digests
	Size         int64             `json:"size"`         // Image size in bytes
	VirtualSize  int64             `json:"virtual_size"` // Virtual size including shared layers
	CreatedAt    time.Time         `json:"created_at"`   // Image creation time
	Labels       map[string]string `json:"labels"`
	Architecture string            `json:"architecture"` // amd64, arm64, etc.
	OS           string            `json:"os"`           // linux, windows
	// Layer information
	Layers []string `json:"layers"` // Layer IDs (sha256 digests)
	// Configuration
	Config  *ImageConfig   `json:"config,omitempty"`
	History []ImageHistory `json:"history,omitempty"`
}

// ImageConfig contains image configuration details
type ImageConfig struct {
	Hostname     string              `json:"hostname,omitempty"`
	User         string              `json:"user,omitempty"`
	ExposedPorts map[string]struct{} `json:"exposed_ports,omitempty"` // e.g., {"80/tcp": {}}
	Env          []string            `json:"env,omitempty"`
	Cmd          []string            `json:"cmd,omitempty"`
	Entrypoint   []string            `json:"entrypoint,omitempty"`
	WorkingDir   string              `json:"working_dir,omitempty"`
	Volumes      map[string]struct{} `json:"volumes,omitempty"`
	Labels       map[string]string   `json:"labels,omitempty"`
}

// ImageHistory represents a layer in image history
type ImageHistory struct {
	CreatedAt  time.Time `json:"created_at"`
	CreatedBy  string    `json:"created_by"` // Command that created this layer
	Size       int64     `json:"size"`
	Comment    string    `json:"comment,omitempty"`
	EmptyLayer bool      `json:"empty_layer"`
}

// GetMainTag returns the primary tag (first in list or <none> if no tags)
func (i *Image) GetMainTag() string {
	if len(i.RepoTags) == 0 {
		return "<none>"
	}
	return i.RepoTags[0]
}

// HasTag checks if the image has a specific tag
func (i *Image) HasTag(tag string) bool {
	for _, t := range i.RepoTags {
		if t == tag {
			return true
		}
	}
	return false
}

// IsDangling checks if the image is dangling (no tags)
func (i *Image) IsDangling() bool {
	return len(i.RepoTags) == 0 || (len(i.RepoTags) == 1 && i.RepoTags[0] == "<none>:<none>")
}

// GetSizeHumanReadable returns size in human-readable format
func (i *Image) GetSizeHumanReadable() string {
	const unit = 1024
	size := float64(i.Size)
	if size < unit {
		return string(rune(int(size))) + " B"
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return string(rune(int(size/float64(div)))) + " " + "KMGTPE"[exp:exp+1] + "B"
}

// GetLayerCount returns the number of layers
func (i *Image) GetLayerCount() int {
	return len(i.Layers)
}

// GetExposedPortsList returns a list of exposed ports
func (i *Image) GetExposedPortsList() []string {
	if i.Config == nil || i.Config.ExposedPorts == nil {
		return []string{}
	}
	ports := make([]string, 0, len(i.Config.ExposedPorts))
	for port := range i.Config.ExposedPorts {
		ports = append(ports, port)
	}
	return ports
}
