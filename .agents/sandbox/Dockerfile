# Sandbox container for Asimi agent shell commands
FROM golang:1.25.3-bookworm

# Install common development tools
RUN apt-get update && apt-get install -y \
    git \
    curl \
    ripgrep \
    pkg-config \
    libgpgme-dev \
    libbtrfs-dev \
    && rm -rf /var/lib/apt/lists/*

# Install just task runner
RUN curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin

# Default to bash
CMD ["/bin/bash"]
