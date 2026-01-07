FROM golang:1.25-alpine AS build
RUN apk add --no-cache git
COPY . /app
WORKDIR /app/
RUN cd ./cmd/ \
    && go build  -ldflags="-s -w -X main.version=$(git rev-parse HEAD)" -o kitchenbot

FROM golang:1.25-alpine AS run
RUN mkdir -p /app
COPY --from=build /app/cmd/kitchenbot /app/kitchenbot
WORKDIR /app
CMD ["./kitchenbot"]