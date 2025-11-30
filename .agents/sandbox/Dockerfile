FROM golang:1.25-bookworm

# Install common development tools and C libraries for Go dependencies
RUN apt-get update && apt-get install -y \
    git \
    curl \
    ripgrep \
    libgpgme-dev \
    libbtrfs-dev \
    libdevmapper-dev \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

# Install just task runner
RUN curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin

COPY .agents/sandbox/bashrc /root/.bashrc
CMD ["/bin/bash"]
