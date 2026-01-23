FROM node:24-alpine AS frontend-build
WORKDIR /app/frontend
COPY ./frontend/package*.json ./
RUN npm install
COPY ./frontend ./
RUN npm run build

FROM golang:1-alpine AS backend-build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
COPY internal/ ./internal/
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Stage 3: Final image
FROM alpine:latest

WORKDIR /app

# Copy backend binary
COPY --from=backend-build /app/server ./server

# Copy frontend build
COPY --from=frontend-build /app/frontend/dist ./frontend/dist

EXPOSE 8080

ENTRYPOINT ["./server"]
