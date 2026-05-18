# Bilibili to VRChat Streamer

Small Go service that accepts a Bilibili URL, downloads it with `yt-dlp`, remuxes it with `ffmpeg`, and exposes a VRChat-friendly `.m3u8` or `.mp4` URL.

## Requirements

- Go 1.22+
- `yt-dlp`
- `ffmpeg`

On Ubuntu:

```bash
sudo apt update
sudo apt install -y ffmpeg python3-pip
python3 -m pip install -U yt-dlp
```

## Run

```bash
go run ./cmd/server
```

Open:

```text
http://localhost:8090
```

Create a job with curl:

```bash
curl -X POST http://localhost:8090/api/jobs \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://www.bilibili.com/video/BVxxxx","format":"hls"}'
```

Poll the returned `status_url`. When the job is ready, paste `playback_url` into a VRChat video player.

## Configuration

Environment variables:

| Name | Default | Description |
| --- | --- | --- |
| `ADDR` | `:8090` | HTTP listen address. |
| `PUBLIC_BASE_URL` | `http://localhost:8090` | Public URL used to build playback links. Set this to your HTTPS domain behind Nginx. |
| `DATA_DIR` | `data` | Job metadata and generated media directory. |
| `YTDLP_PATH` | `yt-dlp` | Path to yt-dlp. |
| `YTDLP_COOKIES_FILE` | empty | Optional Netscape-format cookies file for videos that need login. |
| `YTDLP_COOKIES_FROM_BROWSER` | empty | Optional browser cookie source, such as `chrome`, `edge`, or `firefox`. `YTDLP_COOKIES_FILE` takes priority. |
| `YTDLP_REFERER` | `https://www.bilibili.com/` | Referer passed to yt-dlp. |
| `YTDLP_USER_AGENT` | desktop Chrome UA | User-Agent passed to yt-dlp. |
| `YTDLP_EXTRA_ARGS` | empty | Optional space-separated extra arguments appended before the source URL. |
| `FORMAT_SELECTOR` | `bv*[vcodec^=avc1]+ba/b[vcodec^=avc1]/bv*+ba/b` | yt-dlp format selector. Defaults to H.264-first output for better VRChat compatibility. |
| `FFMPEG_PATH` | `ffmpeg` | Path to ffmpeg. |
| `MAX_CONCURRENT_JOBS` | `1` | Concurrent conversion jobs. |
| `JOB_TIMEOUT_MINUTES` | `90` | Per command timeout. |
| `ALLOWED_HOSTS` | `bilibili.com,www.bilibili.com,m.bilibili.com,b23.tv` | Comma-separated allowed source hosts. |

## Nginx Example

```nginx
server {
    listen 443 ssl http2;
    server_name vrc-video.example.com;

    client_max_body_size 20m;

    location / {
        proxy_pass http://127.0.0.1:8090;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

Run the service with:

```bash
PUBLIC_BASE_URL=https://vrc-video.example.com ./bili-vrc-streamer
```

## Notes

- Bilibili can return HTTP 412 or require cookies even for public videos. Export a Netscape-format cookies file and set `YTDLP_COOKIES_FILE=/path/to/cookies.txt`. You can also try `YTDLP_COOKIES_FROM_BROWSER=chrome`, `edge`, or `firefox`, but this can fail while the browser profile is locked.
- Generated HLS/MP4 files consume disk space. Put `DATA_DIR` on a volume with enough storage and add a cleanup timer when this becomes long-running infrastructure.
- Only process videos you have the right to play or share.
