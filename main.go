package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/siderolabs/talos/pkg/machinery/client"
	clientconfig "github.com/siderolabs/talos/pkg/machinery/client/config"
	"google.golang.org/protobuf/types/known/emptypb"
)

type TalosMCP struct {
	talosClient *client.Client
}

func NewTalosMCP() *TalosMCP {
	return &TalosMCP{}
}

func (m *TalosMCP) getTalosClient() (*client.Client, error) {

	clientConfig, err := clientconfig.Open("/Users/qjoly/code/mcp-talos/talosconfig")
	if err != nil {
		return nil, fmt.Errorf("error when opening talos config file: %w", err)
	}

	ctx := client.WithNodes(context.TODO(), "192.168.32.83,192.168.32.84")

	opts := []client.OptionFunc{
		client.WithConfig(clientConfig),
		client.WithConfigFromFile("/Users/qjoly/code/mcp-talos/talosconfig"),
		client.WithEndpoints("192.168.32.83"),
	}

	m.talosClient, err = client.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("error when instanciating talos client: %w", err)
	}

	return m.talosClient, nil
}

func (m *TalosMCP) ListDisks(ctx context.Context) ([]map[string]interface{}, error) {

	talosClient, err := m.getTalosClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Talos client: %w", err)
	}

	listNodes := []string{"192.168.32.83", "192.168.32.84"}
	var disks []map[string]interface{}
	for _, node := range listNodes {
		resp, err := talosClient.Disks(client.WithNodes(context.TODO(), node))
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

	talosClient, err := m.getTalosClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Talos client: %w", err)
	}

	resp, err := talosClient.MachineClient.NetworkDeviceStats(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("failed to list network interfaces: %w", err)
	}

	var interfaces []map[string]interface{}
	for _, msg := range resp.Messages {
		for _, iface := range msg.Devices {
			interfaces = append(interfaces, map[string]interface{}{
				"name":    iface.Name,
				"TxBytes": iface.TxBytes,
				"RxBytes": iface.RxBytes,
			})
		}
	}

	return interfaces, nil
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

	// Initialize Talos MCP
	talosMCP := NewTalosMCP()

	// Add disk listing tool
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

	// Add network interfaces listing tool
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

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
