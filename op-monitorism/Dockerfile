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

# Install ca-certificates so that HTTPS works
RUN apk update && apk add --no-cache ca-certificates

# Copy the built binary from the builder stage
COPY --from=builder /app/bin/monitorism /usr/local/bin/monitorism

# Ensure the binary is executable
RUN chmod +x /usr/local/bin/monitorism

# Set the command to run the binary
CMD ["monitorism"]
