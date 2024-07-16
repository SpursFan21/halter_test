# Use the official Golang image to create a build environment.
# This is based on Debian and sets the GOPATH to /go.
FROM golang:1.22.5 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go build -o writer ./writer

# Use a smaller base image for resulting container
FROM golang:1.22.5

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/writer /app/writer

# Command to run the executable
CMD ["./writer"]
