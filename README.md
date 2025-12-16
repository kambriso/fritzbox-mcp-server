# FRITZ!Box MCP Server

Model Context Protocol (MCP) server for FRITZ!Box routers.

This repository contains release binaries. For source code, documentation, and issues, see: https://github.com/kambriso/fritz-mcp

## Installation

Download the latest release for your platform from the [Releases](https://github.com/kambriso/fritzbox-mcp-server/releases) page.

### Linux / macOS

```bash
# Download and extract
curl -L https://github.com/kambriso/fritzbox-mcp-server/releases/latest/download/fritz-mcp-linux-amd64.tar.xz | tar xJ

# Make executable and move to PATH
chmod +x fritz-mcp
sudo mv fritz-mcp /usr/local/bin/
```

### Windows

Download `fritz-mcp-windows-amd64.zip`, extract, and run `fritz-mcp.exe`.

## Configuration

Create `~/.config/fritz-mcp/.env`:

```
FRITZ_HOST=fritz.box
FRITZ_PORT=49000
FRITZ_USERNAME=your-username
FRITZ_PASSWORD=your-password
```

## License

Business Source License 1.1 - See [LICENSE](LICENSE) for details.

Change License: Apache License 2.0 (effective 2028-11-25)
