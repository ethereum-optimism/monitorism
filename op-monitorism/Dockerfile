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

# Use WizOS base image instead of Alpine
FROM registry.os.wiz.io/wizos-base:latest

# Update package index and install ca-certificates
# WizOS uses apk package manager like Alpine
RUN apk update && apk add --no-cache ca-certificates

# Copy the application binary
COPY /app/bin/monitorism /usr/local/bin/monitorism

# Make the binary executable
RUN chmod +x /usr/local/bin/monitorism

# Set the command to run the application
CMD ["monitorism"]
