package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/client"
	clientconfig "github.com/siderolabs/talos/pkg/machinery/client/config"
	"google.golang.org/protobuf/types/known/emptypb"
)

type TalosMCP struct {
	talosClient *client.Client
	nodes       []string
	endpoint    string
}

func NewTalosMCP() *TalosMCP {
	return &TalosMCP{}
}

func (m *TalosMCP) setTalosClient() (*client.Client, error) {

	clientConfig, err := clientconfig.Open("/Users/qjoly/code/mcp-talos/talosconfig")
	if err != nil {
		return nil, fmt.Errorf("error when opening talos config file: %w", err)
	}

	ctx := client.WithNodes(context.TODO(), m.nodes...)

	configPath := os.Getenv("TALOSCONFIG")
	if configPath == "" {
		configPath = "/Users/qjoly/code/mcp-talos/talosconfig"
	}

	m.endpoint = clientConfig.Contexts[clientConfig.Context].Endpoints[0]
	m.nodes = clientConfig.Contexts[clientConfig.Context].Nodes

	opts := []client.OptionFunc{
		client.WithConfigFromFile(configPath),
		client.WithConfig(clientConfig),
		client.WithEndpoints(m.endpoint),
	}

	m.talosClient, err = client.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("error when instanciating talos client: %w", err)
	}

	return m.talosClient, nil
}

func (m *TalosMCP) ListDisks(ctx context.Context) ([]map[string]interface{}, error) {

	if m.talosClient == nil {
		m.setTalosClient()
	}

	var disks []map[string]interface{}
	for _, node := range m.nodes {
		resp, err := m.talosClient.Disks(client.WithNodes(context.TODO(), node))
		if err != nil {
			return nil, fmt.Errorf("failed to list disks for node %s: %w", node, err)
		}

		nodeDisks := map[string]interface{}{
			"node":  node,
			"disks": []map[string]interface{}{},
		}

		for _, msg := range resp.Messages {
			for _, disk := range msg.Disks {
				nodeDisks["disks"] = append(nodeDisks["disks"].([]map[string]interface{}), map[string]interface{}{
					"device": disk.DeviceName,
					"model":  disk.Model,
					"size":   disk.Size,
					"type":   disk.Type.String(),
					"uuid":   disk.Uuid,
				})
			}
		}

		disks = append(disks, nodeDisks)
	}

	return disks, nil
}

func (m *TalosMCP) ListNetworkInterfaces(ctx context.Context) ([]map[string]interface{}, error) {

	if m.talosClient == nil {
		_, err := m.setTalosClient()
		if err != nil {
			return nil, fmt.Errorf("error when instanciating talos client: %w", err)
		}
	}

	var interfaces []map[string]interface{}
	for _, node := range m.nodes {
		resp, err := m.talosClient.MachineClient.NetworkDeviceStats(client.WithNodes(ctx, node), &emptypb.Empty{})
		if err != nil {
			return nil, fmt.Errorf("failed to list network interfaces for node %s: %w", node, err)
		}

		for _, msg := range resp.Messages {
			for _, iface := range msg.Devices {
				interfaces = append(interfaces, map[string]interface{}{
					"node":    node,
					"name":    iface.Name,
					"TxBytes": iface.TxBytes,
					"RxBytes": iface.RxBytes,
				})
			}
		}
	}

	return interfaces, nil
}

func (m *TalosMCP) ListMemory(ctx context.Context) ([]map[string]interface{}, error) {

	if m.talosClient == nil {
		_, err := m.setTalosClient()
		if err != nil {
			return nil, fmt.Errorf("error when instanciating talos client: %w", err)
		}
	}

	var memoryInfo []map[string]interface{}
	for _, node := range m.nodes {
		resp, err := m.talosClient.MachineClient.Memory(client.WithNodes(ctx, node), &emptypb.Empty{})
		if err != nil {
			return nil, fmt.Errorf("failed to list memory for node %s: %w", node, err)
		}

		for _, msg := range resp.Messages {
			memoryInfo = append(memoryInfo, map[string]interface{}{
				"node":    node,
				"Meminfo": msg.Meminfo,
			})
		}
	}

	return memoryInfo, nil
}

func (m *TalosMCP) ListCPU(ctx context.Context) ([]map[string]interface{}, error) {

	if m.talosClient == nil {
		_, err := m.setTalosClient()
		if err != nil {
			return nil, fmt.Errorf("error when instanciating talos client: %w", err)
		}
	}

	var cpuInfo []map[string]interface{}
	for _, node := range m.nodes {
		resp, err := m.talosClient.MachineClient.CPUInfo(client.WithNodes(ctx, node), &emptypb.Empty{})
		if err != nil {
			return nil, fmt.Errorf("failed to list CPU usage for node %s: %w", node, err)
		}

		for _, msg := range resp.Messages {
			cpuInfo = append(cpuInfo, map[string]interface{}{
				"node": node,
				"CPU":  msg.CpuInfo,
			})
		}
	}

	return cpuInfo, nil
}

func (m *TalosMCP) RebootNode(ctx context.Context, node string) error {
	if m.talosClient == nil {
		_, err := m.setTalosClient()
		if err != nil {
			return fmt.Errorf("error when instanciating talos client: %w", err)
		}
	}

	fmt.Printf("Rebooting node %s...\n", node)

	_, err := m.talosClient.MachineClient.Reboot(client.WithNodes(ctx, node), &machine.RebootRequest{
		Mode: 1,
	})
	if err != nil {
		return fmt.Errorf("failed to reboot node %s: %w", node, err)
	}

	return nil
}

func main() {
	// Create a new MCP server
	s := server.NewMCPServer(
		"Talos Cluster Manager",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
		server.WithRecovery(),
	)

	talosMCP := NewTalosMCP()

	diskTool := mcp.NewTool("list_disks",
		mcp.WithDescription("List all disks in the Talos cluster"),
	)

	s.AddTool(diskTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

		disks, err := talosMCP.ListDisks(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list disks: %v", err)), nil
		}

		disksJSON, err := json.MarshalIndent(disks, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal disks: %v", err)), nil
		}

		return mcp.NewToolResultText(string(disksJSON)), nil
	})

	networkTool := mcp.NewTool("list_network_interfaces",
		mcp.WithDescription("List all network interfaces in the Talos cluster"),
	)

	s.AddTool(networkTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

		interfaces, err := talosMCP.ListNetworkInterfaces(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list network interfaces: %v", err)), nil
		}

		interfacesJSON, err := json.MarshalIndent(interfaces, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal interfaces: %v", err)), nil
		}

		return mcp.NewToolResultText(string(interfacesJSON)), nil
	})

	memoryTool := mcp.NewTool("list_memory",
		mcp.WithDescription("List memory information for all nodes in the Talos cluster"),
	)

	s.AddTool(memoryTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

		memory, err := talosMCP.ListMemory(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list memory: %v", err)), nil
		}

		memoryJSON, err := json.MarshalIndent(memory, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal memory: %v", err)), nil
		}

		return mcp.NewToolResultText(string(memoryJSON)), nil
	})

	cpuTool := mcp.NewTool("list_cpu",
		mcp.WithDescription("List CPU usage information for all nodes in the Talos cluster"),
	)

	s.AddTool(cpuTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

		cpu, err := talosMCP.ListCPU(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list CPU usage: %v", err)), nil
		}

		cpuJSON, err := json.MarshalIndent(cpu, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal CPU usage: %v", err)), nil
		}

		return mcp.NewToolResultText(string(cpuJSON)), nil
	})

	rebootTool := mcp.NewTool("reboot_node",
		mcp.WithDescription("Reboot a specific node in the Talos cluster"),
		mcp.WithString("node"),
	)

	s.AddTool(rebootTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		node, ok := request.Params.Arguments["node"].(string)
		if !ok || node == "" {
			return mcp.NewToolResultError("missing or invalid 'node' argument"), nil
		}

		err := talosMCP.RebootNode(ctx, node)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to reboot node: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Node %s rebooted successfully", node)), nil
	})

	if os.Getenv("MCP_SERVER") != "" {
		message, err := talosMCP.ListNetworkInterfaces(context.Background())
		if err != nil {
			fmt.Printf("Error listing network interfaces: %v\n", err)
			return
		}
		fmt.Printf("Network interfaces: %v\n", message)
		message, err = talosMCP.ListDisks(context.Background())
		if err != nil {
			fmt.Printf("Error listing disks: %v\n", err)
		}
		fmt.Printf("Disks: %v\n", message)
		fmt.Println("MCP server started")

		return
	}

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
