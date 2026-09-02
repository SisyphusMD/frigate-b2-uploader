# Start from the official Go image to build our application.
FROM golang:1.27.1@sha256:137ca8442e368f5f5bb4d4f84d8cc1f6c2d898edf8a380459eb407f89d149b0d AS builder

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
FROM alpine:latest  
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from the builder stage.
COPY --from=builder /app/main .

# Command to run the executable.
CMD ["./main"]
