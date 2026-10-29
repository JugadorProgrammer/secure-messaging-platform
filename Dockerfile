FROM ubuntu:24.04 AS builder

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        build-essential \
        cmake \
        pkg-config \
        libdrogon-dev \
        libpq-dev \
        libjsoncpp-dev \
        uuid-dev \
        zlib1g-dev \
        libsqlite3-dev \
        libmariadb-dev \
        libmariadb-dev-compat \
        libbrotli-dev \
        libhiredis-dev \
        libyaml-cpp-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY CMakeLists.txt ./
COPY include/ ./include/
COPY src/ ./src/

RUN cmake -S . -B build \
        -DCMAKE_BUILD_TYPE=Release \
    && cmake --build build --parallel 2


# Stage 2: runtime environment
FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        libdrogon1t64 \
        libpq5 \
        libjsoncpp25 \
        libssl3t64 \
        ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/build/smp_auth /app/smp_auth
COPY config/ /app/config/

EXPOSE 8080

CMD ["./smp_auth"]
