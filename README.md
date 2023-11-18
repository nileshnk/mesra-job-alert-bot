# BIT Mesra Telegram Job Alert Bot

A simple alert bot which scrapes Bit Mesra's Job Portal for new Posts. It sends a Post in Telegram channel whenever New Post is listed.

## Installation

### Local

```
go run .
```

### Docker

- Build the image

```
docker build -t job-alert-bot .
```

- Run the image

```
docker run -d --env-file ./.env --name Mesra-Job-Alert job-alert-bot
```
