# Asimi Sandbox Container
# Debian-based container with Go runtime and development tools

FROM golang:1.25-bookworm

# Install common development tools and build dependencies
RUN apt-get update && apt-get install -y \
    git \
    curl \
    ripgrep \
    libgpgme-dev \
    libdevmapper-dev \
    libbtrfs-dev \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

# Install just command runner
RUN curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin

# Create asimi user and setup environment
RUN useradd -m -s /bin/bash asimi
COPY .agents/sandbox/bashrc /home/asimi/.bashrc
USER asimi
CMD ["/bin/bash"]
