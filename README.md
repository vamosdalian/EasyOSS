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
| `-port` | 9000 | Server port |
| `-data` | ./data | Data storage path |
| `-version` | - | Show version information |

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

```yaml
version: '3.8'
services:
  easyoss:
    image: ghcr.io/vamosdalian/easyoss:latest
    ports:
      - "9000:9000"
    volumes:
      - ./data:/data
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
