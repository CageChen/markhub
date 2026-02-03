# Getting Started

This guide will help you get started with MarkHub.

## Installation

### From Source

```bash
git clone https://github.com/CageChen/markhub.git
cd markhub
make build
./bin/markhub serve --path ./docs
```

### Using Docker

```bash
docker run -p 8080:8080 -v $(pwd)/docs:/docs markhub
```

## Usage

Basic usage:

```bash
markhub serve --path ./docs --port 8080
```

## Configuration

Create a `markhub.yaml` file:

```yaml
path: ./docs
port: 8080
theme: dark
watch: true
```

## Next Steps

- Read the [API Documentation](api.md)
- Check out [Examples](examples/index.md)
