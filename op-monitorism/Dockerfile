# Define the build stage
FROM chainguard/wolfi-base:latest AS builder

# Set the working directory inside the container
WORKDIR /app

# Install system dependencies including 'make' and the Go toolchain
RUN apk add --no-cache go-1.26 make build-base git

# Copy the source code and Makefile into the container
COPY . .

# Run the Makefile command to build the binary
RUN make

# Define the final base image
FROM chainguard/wolfi-base:latest

# Install ca-certificates so that HTTPS works
RUN apk add --no-cache ca-certificates

# Copy the built binary from the builder stage
COPY --from=builder /app/bin/monitorism /usr/local/bin/monitorism

# Set the command to run the binary
CMD ["monitorism"]
