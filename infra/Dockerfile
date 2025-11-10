# Asimi Development Container
# Base image: Go runtime on Debian
FROM golang:1.25-bookworm

# Install common development tools and build dependencies
RUN apt-get update && apt-get install -y \
    git \
    curl \
    ripgrep \
    vim \
    libgpgme-dev \
    libbtrfs-dev \
    && rm -rf /var/lib/apt/lists/*

# Install just command runner
RUN curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin

# Install golangci-lint
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /usr/local/bin

# Set default shell to bash
CMD ["/bin/bash"]
