# Define the build stage
FROM dhi.io/golang:1.24-alpine3.22-dev AS builder

# Set the working directory inside the container
WORKDIR /app

# Install system dependencies including 'make'
RUN apk update && apk add --no-cache make

# Copy the source code and Makefile into the container
COPY . .

# Run the Makefile command to build the binary
RUN make

# Ensure the binary is executable
RUN chmod +x /app/bin/defender

# Define the final base image
FROM dhi.io/alpine-base:3.22

# Copy the built binary from the builder stage
COPY --from=builder /app/bin/defender /usr/local/bin/defender

# Set the command to run the binary
CMD ["defender"]
