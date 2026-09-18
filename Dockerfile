# Start from the official Go image to build our application.
FROM golang:1.22.2@sha256:d5302d40dc5fbbf38ec472d1848a9d2391a13f93293a6a5b0b87c99dc0eaa6ae AS builder

# Set the current working directory inside the container.
WORKDIR /app

# Copy the go mod and sum files to download the dependencies.
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# Copy the rest of the application's source code.
COPY . .

# Build the application.
RUN CGO_ENABLED=0 GOOS=linux go build -v -o main .

# Start a new stage from scratch for a smaller, final image.
FROM alpine:latest@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6  
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from the builder stage.
COPY --from=builder /app/main .

# Command to run the executable.
CMD ["./main"]
