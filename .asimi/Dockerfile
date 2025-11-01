# Development container for Asimi CLI
FROM golang:1.25.3-bookworm

# Install common development tools
RUN apt-get update && apt-get install -y \
    git \
    curl \
    ripgrep \
    vim \
    less \
    && rm -rf /var/lib/apt/lists/*

# Install just task runner
RUN curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin

# Install golangci-lint
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /usr/local/bin

# Set up Go environment
ENV GO111MODULE=on \
    CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64

# Default command
CMD ["/bin/bash"]
