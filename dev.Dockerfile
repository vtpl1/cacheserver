FROM ubuntu:20.04

RUN apt update \
    && DEBIAN_FRONTEND=noninteractive apt install -y \
    ca-certificates \
    software-properties-common \
    make \
    wget \
    curl \
    git \
    unzip \
    python3-pip \
    lsb-release && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir debs && \
    dpkg --purge --force-remove-reinstreq mongodb-database-tools && \
    wget -q https://fastdl.mongodb.org/tools/db/mongodb-database-tools-ubuntu2204-x86_64-100.9.4.deb -P ./debs && \
    apt-get install -y -q --no-install-recommends ./debs/*.deb && \
    rm -rf debs && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

RUN curl -sSL "https://repo.mysql.com/apt/ubuntu/pool/mysql-apt-config/m/mysql-apt-config/mysql-apt-config_0.8.29-1_all.deb" -o "mysql-apt-config.deb" && \
    export DEBIAN_FRONTEND=noninteractive && \
    dpkg -i "mysql-apt-config.deb" && \
    apt-get update && \
    apt-get install -y mysql-shell && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

ARG GO_VERSION=1.23.3

RUN mkdir gotmp && \
    wget -O go.linux-amd64.tar.gz https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz && \
    rm -rf /usr/local/go && tar -C /usr/local -xzf go.linux-amd64.tar.gz && \
    rm -rf gotmp

ENV PATH="${PATH}:/usr/local/go/bin"

# RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

ENV SHELL=/bin/bash
ENV LANG=en_US.utf-8
ENV LC_ALL=en_US.utf-8

ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=$USER_UID

# Create the user
RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME

RUN groupmod --gid $USER_GID $USERNAME \
    && usermod --uid $USER_UID --gid $USER_GID $USERNAME \
    && chown -R $USER_UID:$USER_GID /home/$USERNAME

USER ${USERNAME}

ARG NODE_VERSION=18

RUN wget -q -O - https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash \
    && export NVM_DIR="$HOME/.nvm" \
    && [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh" \
    && nvm install ${NODE_VERSION}

ENV PB_REL="https://github.com/protocolbuffers/protobuf/releases"
ENV PROTOC_VERSION=28.3
RUN mkdir -p "/home/$USERNAME/tmp" \
    && wget -O "/home/$USERNAME/tmp/protoc-linux-x86_64.zip" "$PB_REL/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip" \
    && unzip "/home/$USERNAME/tmp/protoc-linux-x86_64.zip" -d "/home/$USERNAME/.local" \
    && rm -rf "/home/$USERNAME/tmp"

ENV PATH="$PATH:/home/$USERNAME/.local/bin"

# RUN python -m poetry self add poetry-bumpversion
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

RUN go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
RUN go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

RUN go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

RUN BIN="/home/$USERNAME/.local/bin" && \
    VERSION="1.46.0" && \
    curl -sSL \
    "https://github.com/bufbuild/buf/releases/download/v${VERSION}/buf-$(uname -s)-$(uname -m)" \
    -o "${BIN}/buf" && \
    chmod +x "${BIN}/buf"

ENV PATH="$PATH:/home/$USERNAME/go/bin"

RUN pip install -U pip poetry bump2version