# EasyOSS

An Easy Object Storage Service like MinIO - S3 compatible, lightweight, and simple to deploy.

## Features

- **S3 Compatible API**: Works with any S3 client (AWS SDK, MinIO Client, etc.)
- **Simple Web UI**: Browse buckets and objects through a web interface
- **Local Storage**: Uses local file system for data and metadata storage
- **Docker Support**: Easy deployment with Docker
- **Multi-Platform**: Supports amd64 and arm64 architectures

## Quick Start

### Using Docker

```bash
docker run -d -p 9000:9000 -v ./data:/data ghcr.io/vamosdalian/easyoss:latest
```

### Building from Source

```bash
# Clone the repository
git clone https://github.com/vamosdalian/EasyOSS.git
cd EasyOSS

# Build
go build -o easyoss ./cmd/easyoss

# Run
./easyoss -port 9000 -data ./data
```

## Usage

### Web UI

Open your browser and navigate to `http://localhost:9000/web/` to access the web interface.

### S3 API

The S3 API is available at `http://localhost:9000/`. You can use any S3 client to interact with it.

#### Example with AWS CLI

```bash
# Configure AWS CLI (use any access key/secret)
aws configure
# AWS Access Key ID: any
# AWS Secret Access Key: any
# Default region name: us-east-1

# Create a bucket
aws --endpoint-url http://localhost:9000 s3 mb s3://mybucket

# Upload a file
aws --endpoint-url http://localhost:9000 s3 cp myfile.txt s3://mybucket/

# List buckets
aws --endpoint-url http://localhost:9000 s3 ls

# List objects in bucket
aws --endpoint-url http://localhost:9000 s3 ls s3://mybucket/

# Download a file
aws --endpoint-url http://localhost:9000 s3 cp s3://mybucket/myfile.txt ./downloaded.txt

# Delete a file
aws --endpoint-url http://localhost:9000 s3 rm s3://mybucket/myfile.txt

# Delete a bucket
aws --endpoint-url http://localhost:9000 s3 rb s3://mybucket
```

## Command Line Options

| Option | Default | Description |
|--------|---------|-------------|
| `-port` | - | Combined port for both S3 API and Web UI (overrides `-s3-port` and `-web-port`) |
| `-s3-port` | 9000 | S3 API port |
| `-web-port` | 9001 | Web UI port |
| `-data` | ./data | Data storage path |
| `-meta` | - | Metadata storage path (defaults to `<data>/.meta`) |
| `-version` | - | Show version information |

## Environment Variables

Environment variables can override command line options:

| Variable | Description |
|----------|-------------|
| `EASYOSS_PORT` | Combined port for both S3 API and Web UI |
| `EASYOSS_S3_PORT` | S3 API port |
| `EASYOSS_WEB_PORT` | Web UI port |
| `EASYOSS_DATA_PATH` | Data storage path |
| `EASYOSS_META_PATH` | Metadata storage path |

Priority order: Environment variables > Command line options > Defaults

## Supported S3 Operations

| Operation | Endpoint | Method |
|-----------|----------|--------|
| ListBuckets | `/` | GET |
| CreateBucket | `/{bucket}` | PUT |
| DeleteBucket | `/{bucket}` | DELETE |
| HeadBucket | `/{bucket}` | HEAD |
| ListObjects | `/{bucket}` | GET |
| GetObject | `/{bucket}/{key}` | GET |
| PutObject | `/{bucket}/{key}` | PUT |
| DeleteObject | `/{bucket}/{key}` | DELETE |
| HeadObject | `/{bucket}/{key}` | HEAD |

## Docker Compose Example

### Combined Mode (Single Port)

```yaml
version: '3.8'
services:
  easyoss:
    image: ghcr.io/vamosdalian/easyoss:latest
    ports:
      - "9000:9000"
    volumes:
      - ./data:/data
    environment:
      - EASYOSS_PORT=9000
    restart: unless-stopped
```

### Separate Ports Mode

```yaml
version: '3.8'
services:
  easyoss:
    image: ghcr.io/vamosdalian/easyoss:latest
    ports:
      - "9000:9000"  # S3 API
      - "9001:9001"  # Web UI
    volumes:
      - ./data:/data
      - ./meta:/meta
    environment:
      - EASYOSS_S3_PORT=9000
      - EASYOSS_WEB_PORT=9001
      - EASYOSS_DATA_PATH=/data
      - EASYOSS_META_PATH=/meta
    restart: unless-stopped
```

## Development

### Prerequisites

- Go 1.23 or later

### Build

```bash
go build -o easyoss ./cmd/easyoss
```

### Test

```bash
go test -v ./...
```

### Build Docker Image

```bash
docker build -t easyoss .
```

## License

MIT License - see [LICENSE](LICENSE) file for details.
