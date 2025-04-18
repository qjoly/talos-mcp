package main

import (
	"context"
	"encoding/json"
	"fmt"

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

func (m *TalosMCP) setTalosClient() error {
	clientConfig, err := clientconfig.Open("talosconfig")
	if err != nil {
		return fmt.Errorf("error when opening talos config file: %w", err)
	}

	opts := []client.OptionFunc{
		client.WithConfig(clientConfig),
	}

	m.talosClient, err = client.New(context.TODO(), opts...)
	if err != nil {
		return fmt.Errorf("error when instanciating talos client: %w", err)
	}

	return nil
}

func (m *TalosMCP) ListDisks(ctx context.Context) ([]map[string]interface{}, error) {
	if m.talosClient == nil {
		return nil, fmt.Errorf("not connected to Talos")
	}

	resp, err := m.talosClient.Disks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list disks: %w", err)
	}

	var disks []map[string]interface{}
	for _, msg := range resp.Messages {
		for _, disk := range msg.Disks {
			disks = append(disks, map[string]interface{}{
				"device": disk.DeviceName,
				"model":  disk.Model,
				"size":   disk.Size,
				"type":   disk.Type.String(),
				"uuid":   disk.Uuid,
			})
		}
	}

	return disks, nil
}

func (m *TalosMCP) ListNetworkInterfaces(ctx context.Context) ([]map[string]interface{}, error) {
	if m.talosClient == nil {
		return nil, fmt.Errorf("not connected to Talos")
	}

	resp, err := m.talosClient.MachineClient.NetworkDeviceStats(ctx, &emptypb.Empty{})
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
		if talosMCP.talosClient == nil {
			if err := talosMCP.setTalosClient(); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to connect to Talos: %v", err)), nil
			}
		}

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
		if talosMCP.talosClient == nil {
			if err := talosMCP.setTalosClient(); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to connect to Talos: %v", err)), nil
			}
		}

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

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
