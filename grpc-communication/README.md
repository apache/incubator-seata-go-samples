# Seata GRPC Communication Samples

This directory contains samples demonstrating GRPC communication between Seata clients and Seata server.

## Overview

These samples showcase different aspects of GRPC communication in Seata, from basic connectivity to advanced features like load balancing, streaming, and monitoring.

## Directory Structure

```
grpc-communication/
├── basic/                    # Basic GRPC communication examples
│   ├── client/              # Simple GRPC client
│   └── config/              # Configuration files
├── advanced/                # Advanced GRPC features (TODO)
│   ├── load-balancing/      # Load balancing strategies
│   ├── monitoring/          # Monitoring and metrics
│   └── streaming/           # Streaming communication
├── client-server/           # Full client-server examples (TODO)
│   ├── client/              # Advanced client implementations
│   ├── server/              # Server-side examples
│   └── proto/               # Protocol definitions
└── docs/                    # Documentation and guides
```

## Current Status

### ✅ Implemented
- **Basic GRPC Client**: Demonstrates fundamental GRPC communication setup
- **Configuration Templates**: Shows how to configure GRPC protocol
- **Project Structure**: Complete directory layout for future expansion

### 🚧 Planned (TODO - Waiting for GRPC feature merge)
- **Advanced Load Balancing**: Multiple load balancing strategies
- **Streaming Communication**: Bidirectional streaming examples
- **Connection Pooling**: Efficient connection management
- **Monitoring & Metrics**: Performance monitoring examples
- **Full Client-Server**: Complete end-to-end examples

## Quick Start

### Basic GRPC Communication

1. **Start Seata Server** with GRPC support:
   ```bash
   # Ensure Seata server is configured for GRPC protocol
   # Default GRPC port: 8091
   ```

2. **Run the basic client**:
   ```bash
   cd basic/client
   go run main.go -config=../config/seata-grpc.yml
   ```

### Configuration

The sample uses GRPC protocol configuration in `seata-grpc.yml`:

```yaml
seata:
  transport:
    protocol: "grpc"  # Enable GRPC communication
  service:
    grouplist:
      default: "127.0.0.1:8091"  # Seata server GRPC endpoint
```

## Development Notes

### For Contributors

When the new GRPC features are merged into seata-go, update these areas:

1. **Enable local development**:
   ```bash
   # Uncomment in go.mod for local testing:
   replace seata.apache.org/seata-go => ../seata-go
   ```

2. **Implement advanced features**:
   - Remove TODO comments
   - Add actual implementations in advanced/ directories
   - Update configuration examples

3. **Add new samples**:
   - Follow the established pattern
   - Include configuration files
   - Add documentation

### Current Limitations

- Advanced GRPC features are not yet available in the current seata-go version
- Some configuration options are placeholders for future implementation
- Full streaming and monitoring examples are pending

## Related PRs

- [Link to GRPC implementation PR] (TODO: Add when available)
- [Link to load balancing PR] (TODO: Add when available)
- [Link to monitoring PR] (TODO: Add when available)

## Contributing

When adding new GRPC communication samples:

1. Follow the existing directory structure
2. Include comprehensive configuration examples
3. Add clear documentation and comments
4. Ensure compatibility with the current seata-go version
5. Mark future features clearly with TODO comments

## Requirements

- Go 1.20+
- Seata Server with GRPC support
- seata-go library (see go.mod for version)

## License

Licensed under the Apache License, Version 2.0.