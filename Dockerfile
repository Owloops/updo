# Use the latest version of Golang as the base image
FROM golang:latest

# Set the working directory inside the container
WORKDIR /app

# Copy the Go application source code into the container
COPY . .

# Build the Go application
RUN go build

# Command to run the executable
ENTRYPOINT ["./updo"]
