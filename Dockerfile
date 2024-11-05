FROM golang:1.23.2-alpine

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Create necessary directories
RUN mkdir -p /app/templates /app/attachments

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o main .

# Run the application
CMD ["./main"] 