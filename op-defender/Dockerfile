# Define the build stage
FROM golang:1.22.2-alpine3.19 as builder

# Set the working directory inside the container
WORKDIR /app

# Install system dependencies including 'make'
RUN apk update && apk add --no-cache make

# Copy the source code and Makefile into the container
COPY . .

# Run the Makefile command to build the binary
RUN make

# Define the final base image
FROM alpine:3.18

# Copy the built binary from the builder stage
COPY --from=builder /app/bin/defender /usr/local/bin/defender

# Ensure the binary is executable
RUN chmod +x /usr/local/bin/defender

# Set the command to run the binary
CMD ["defender"]
