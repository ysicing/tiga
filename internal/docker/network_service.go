package docker

import (
	"context"
	"encoding/json"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"

	pb "github.com/ysicing/tiga/pkg/grpc/proto/docker"
)

// ListNetworks implements the ListNetworks RPC method
// It retrieves a list of Docker networks with optional filtering
func (s *DockerService) ListNetworks(ctx context.Context, req *pb.ListNetworksRequest) (*pb.ListNetworksResponse, error) {
	// Parse Docker filters from JSON string
	filterArgs := filters.NewArgs()
	if req.Filters != "" {
		var filterMap map[string][]string
		if err := json.Unmarshal([]byte(req.Filters), &filterMap); err == nil {
			for key, values := range filterMap {
				for _, value := range values {
					filterArgs.Add(key, value)
				}
			}
		}
	}

	// Build Docker API options
	options := types.NetworkListOptions{
		Filters: filterArgs,
	}

	// Call Docker API to list networks
	networks, err := s.dockerClient.Client().NetworkList(ctx, options)
	if err != nil {
		return nil, err
	}

	// Convert Docker network resources to protobuf format
	pbNetworks := make([]*pb.Network, len(networks))
	for i, net := range networks {
		pbNetworks[i] = &pb.Network{
			Id:         net.ID,
			Name:       net.Name,
			Created:    net.Created.Unix(),
			Scope:      net.Scope,
			Driver:     net.Driver,
			EnableIpv6: net.EnableIPv6,
			Internal:   net.Internal,
			Attachable: net.Attachable,
			Ingress:    net.Ingress,
			Labels:     net.Labels,
			Options:    net.Options,
		}

		// Convert IPAM config
		if net.IPAM.Driver != "" {
			ipamConfig := &pb.IPAMConfig{
				Driver:  net.IPAM.Driver,
				Options: net.IPAM.Options,
			}

			// Convert IPAM config entries
			for _, config := range net.IPAM.Config {
				ipamConfig.Config = append(ipamConfig.Config, &pb.IPAMPool{
					Subnet:  config.Subnet,
					Gateway: config.Gateway,
					IpRange: config.IPRange,
				})
			}

			pbNetworks[i].Ipam = ipamConfig
		}

		// Convert containers
		if net.Containers != nil {
			pbNetworks[i].Containers = make(map[string]*pb.NetworkContainer)
			for containerID, endpoint := range net.Containers {
				pbNetworks[i].Containers[containerID] = &pb.NetworkContainer{
					Name:        endpoint.Name,
					EndpointId:  endpoint.EndpointID,
					MacAddress:  endpoint.MacAddress,
					Ipv4Address: endpoint.IPv4Address,
					Ipv6Address: endpoint.IPv6Address,
				}
			}
		}
	}

	return &pb.ListNetworksResponse{
		Networks: pbNetworks,
	}, nil
}

// GetNetwork implements the GetNetwork RPC method
// It retrieves detailed information about a specific Docker network
func (s *DockerService) GetNetwork(ctx context.Context, req *pb.GetNetworkRequest) (*pb.GetNetworkResponse, error) {
	// Inspect the network
	networkResource, err := s.dockerClient.Client().NetworkInspect(ctx, req.NetworkId, types.NetworkInspectOptions{
		Verbose: req.Verbose,
		Scope:   req.Scope,
	})
	if err != nil {
		return nil, err
	}

	// Convert to protobuf format
	pbNetwork := &pb.Network{
		Id:         networkResource.ID,
		Name:       networkResource.Name,
		Created:    networkResource.Created.Unix(),
		Scope:      networkResource.Scope,
		Driver:     networkResource.Driver,
		EnableIpv6: networkResource.EnableIPv6,
		Internal:   networkResource.Internal,
		Attachable: networkResource.Attachable,
		Ingress:    networkResource.Ingress,
		Labels:     networkResource.Labels,
		Options:    networkResource.Options,
	}

	// Convert IPAM config
	if networkResource.IPAM.Driver != "" {
		ipamConfig := &pb.IPAMConfig{
			Driver:  networkResource.IPAM.Driver,
			Options: networkResource.IPAM.Options,
		}

		for _, config := range networkResource.IPAM.Config {
			ipamConfig.Config = append(ipamConfig.Config, &pb.IPAMPool{
				Subnet:  config.Subnet,
				Gateway: config.Gateway,
				IpRange: config.IPRange,
			})
		}

		pbNetwork.Ipam = ipamConfig
	}

	// Convert containers
	if networkResource.Containers != nil {
		pbNetwork.Containers = make(map[string]*pb.NetworkContainer)
		for containerID, endpoint := range networkResource.Containers {
			pbNetwork.Containers[containerID] = &pb.NetworkContainer{
				Name:        endpoint.Name,
				EndpointId:  endpoint.EndpointID,
				MacAddress:  endpoint.MacAddress,
				Ipv4Address: endpoint.IPv4Address,
				Ipv6Address: endpoint.IPv6Address,
			}
		}
	}

	return &pb.GetNetworkResponse{
		Network: pbNetwork,
	}, nil
}

// CreateNetwork implements the CreateNetwork RPC method
// It creates a new Docker network
func (s *DockerService) CreateNetwork(ctx context.Context, req *pb.CreateNetworkRequest) (*pb.CreateNetworkResponse, error) {
	// Build IPAM config
	var ipamConfig *network.IPAM
	if req.Ipam != nil {
		ipamConfig = &network.IPAM{
			Driver:  req.Ipam.Driver,
			Options: req.Ipam.Options,
		}

		for _, pool := range req.Ipam.Config {
			ipamConfig.Config = append(ipamConfig.Config, network.IPAMConfig{
				Subnet:  pool.Subnet,
				IPRange: pool.IpRange,
				Gateway: pool.Gateway,
			})
		}
	}

	// Build network create options
	enableIPv6 := req.EnableIpv6
	createOptions := types.NetworkCreate{
		Driver:     req.Driver,
		Internal:   req.Internal,
		Attachable: req.Attachable,
		Ingress:    req.Ingress,
		EnableIPv6: &enableIPv6,
		IPAM:       ipamConfig,
		Options:    req.Options,
		Labels:     req.Labels,
	}

	// Create the network
	resp, err := s.dockerClient.Client().NetworkCreate(ctx, req.Name, createOptions)
	if err != nil {
		return nil, err
	}

	return &pb.CreateNetworkResponse{
		NetworkId: resp.ID,
		Warning:   resp.Warning,
	}, nil
}

// DeleteNetwork implements the DeleteNetwork RPC method
// It removes a Docker network
func (s *DockerService) DeleteNetwork(ctx context.Context, req *pb.DeleteNetworkRequest) (*pb.DeleteNetworkResponse, error) {
	// Remove the network
	err := s.dockerClient.Client().NetworkRemove(ctx, req.NetworkId)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteNetworkResponse{
		Success: true,
	}, nil
}

// ConnectNetwork implements the ConnectNetwork RPC method
// It connects a container to a Docker network
func (s *DockerService) ConnectNetwork(ctx context.Context, req *pb.ConnectNetworkRequest) (*pb.ConnectNetworkResponse, error) {
	// Build endpoint config
	var endpointConfig *network.EndpointSettings
	if req.EndpointConfig != nil {
		endpointConfig = &network.EndpointSettings{
			Links:   req.EndpointConfig.Links,
			Aliases: req.EndpointConfig.Aliases,
		}

		// Convert IPAM config
		if req.EndpointConfig.IpamConfig != nil {
			endpointConfig.IPAMConfig = &network.EndpointIPAMConfig{}
			if ipv4, ok := req.EndpointConfig.IpamConfig["ipv4_address"]; ok {
				endpointConfig.IPAMConfig.IPv4Address = ipv4
			}
			if ipv6, ok := req.EndpointConfig.IpamConfig["ipv6_address"]; ok {
				endpointConfig.IPAMConfig.IPv6Address = ipv6
			}
		}
	}

	// Connect the container to the network
	err := s.dockerClient.Client().NetworkConnect(ctx, req.NetworkId, req.ContainerId, endpointConfig)
	if err != nil {
		return nil, err
	}

	return &pb.ConnectNetworkResponse{
		Success: true,
	}, nil
}

// DisconnectNetwork implements the DisconnectNetwork RPC method
// It disconnects a container from a Docker network
func (s *DockerService) DisconnectNetwork(ctx context.Context, req *pb.DisconnectNetworkRequest) (*pb.DisconnectNetworkResponse, error) {
	// Disconnect the container from the network
	err := s.dockerClient.Client().NetworkDisconnect(ctx, req.NetworkId, req.ContainerId, req.Force)
	if err != nil {
		return nil, err
	}

	return &pb.DisconnectNetworkResponse{
		Success: true,
	}, nil
}
