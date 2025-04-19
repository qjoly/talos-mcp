# Talos-MCP

:warning: This project is in only a Proof-of-Concept and is not destined to be maintained.

## Description

This is a simple implementation of a Talos MCP (Modele Context Protocol) using the Talos SDK to fetch data from multiple Talos nodes.

Features: 
- List disks
- List network interfaces
- List CPU and memory usage
- Reboot nodes

## Requirements

- Golang 1.24 or higher
- A working Talos cluster

The code is designed to use the endpoint and nodes presents in the `talosconfig` file. You would need to set these values in the config file.

```yaml
context: mcp
contexts:
    mcp:
        endpoints: # These values are mandatory
            - 192.168.32.83
        nodes:
            - 192.168.32.83
            - 192.168.32.85
            - 192.168.32.84
        ca: x
        crt: x
        key: x
```

## Installation

- Clone the repository

```bash
git clone https://github.com/qjoly/talos-mcp.git
```

- Change directory to the project folder

```bash
cd talos-mcp
```

- Build the project

```bash
go build -o talos-mcp main.go
```

- Configure your MCP Client 

The following example is for the MCP client `mcp-copilot` but you can use any MCP client that supports the stdio protocol.

```json
{
    "mcp": {
        "servers": {
            "talos": {
                "type": "stdio",
                "command": "/Users/qjoly/code/mcp-talos/talos-mcp",
                "env": {
                    "TALOSCONFIG": "/Users/qjoly/code/mcp-talos/talosconfig",
                }
            }
        }
    }
}
```
