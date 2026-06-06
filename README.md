# Video to VRChat Streamer

Small Go service that accepts a Bilibili or YouTube URL, downloads it with `yt-dlp`, remuxes it with `ffmpeg`, and exposes a VRChat-friendly `.m3u8` or `.mp4` URL.

When Cloudflare R2 is configured, generated media is uploaded to R2 and the API returns the public R2 URL. The HTTP conversion API requires R2 so server-local video files only exist temporarily during conversion/upload.

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
  -d '{"url":"https://www.youtube.com/watch?v=xxxxxxxxxxx","format":"mp4"}'
```

The `url` field may also contain mobile share text; the service extracts a supported Bilibili or YouTube link before converting.

Poll the returned `status_url`. When the job is ready, paste `playback_url` into a VRChat video player.

With R2 configured, `playback_url` will look like:

```text
https://video.example.com/vrchat/BVxxxx/mp4/video.mp4
https://video.example.com/vrchat/youtube-xxxxxxxxxxx/mp4/video.mp4
```

If R2 is not configured, the HTTP conversion API returns `503 Service Unavailable` instead of downloading media to local disk.

## Direct MP4 Convenience Route

For quick Bilibili MP4 playback, pass `v` on the homepage URL:

```text
http://localhost:8090/?v=BVxxxx
http://localhost:8090/?v=https%3A%2F%2Fwww.bilibili.com%2Fvideo%2FBVxxxx%2F
```

By default, the service resolves the Bilibili HTML5 MP4 URL and proxies the video stream through itself, including range requests from video players. Set `DIRECT_PLAYBACK_MODE=redirect` to switch this route back to returning `302 Found` to Bilibili's temporary `.mp4` link. These Bilibili links still expire upstream, so the generated R2 `playback_url` remains the steadier option for longer sharing.

## Direct Download CLI

Download one video as MP4 and exit:

```bash
go run ./cmd/server "https://www.youtube.com/watch?v=xxxxxxxxxxx"
```

Equivalent explicit form:

```bash
go run ./cmd/server --download "https://www.bilibili.com/video/BVxxxx" --format mp4 --output downloads
```

Generate HLS files instead:

```bash
go run ./cmd/server --download "https://www.bilibili.com/video/BVxxxx" --format hls --output downloads
```

On Windows in this workspace, use the full Go path if `go` is not on PATH:

```powershell
& 'C:\Program Files\Go\bin\go.exe' run ./cmd/server "https://www.bilibili.com/video/BVxxxx"
```

If R2 is configured, the CLI prints both the local downloaded file and the R2 `playback_url`.

## Configuration

Environment variables:

| Name | Default | Description |
| --- | --- | --- |
| `ADDR` | `:8090` | HTTP listen address. |
| `PUBLIC_BASE_URL` | `http://localhost:8090` | Public URL used to build playback links. Set this to your HTTPS domain behind Nginx. |
| `DATA_DIR` | `data` | Job metadata and generated media directory. |
| `ASSETS_DIR` | `web/assets` | Static assets used by the conversion page. |
| `YTDLP_PATH` | `yt-dlp` | Path to yt-dlp. |
| `YTDLP_COOKIES_FILE` | empty | Optional Netscape-format cookies file for videos that need login. |
| `YTDLP_COOKIES_FROM_BROWSER` | empty | Optional browser cookie source, such as `chrome`, `edge`, or `firefox`. `YTDLP_COOKIES_FILE` takes priority. |
| `YTDLP_REFERER` | `https://www.bilibili.com/` | Referer passed to yt-dlp. |
| `YTDLP_USER_AGENT` | desktop Chrome UA | User-Agent passed to yt-dlp. |
| `YTDLP_EXTRA_ARGS` | empty | Optional space-separated extra arguments appended before the source URL. |
| `BILIBILI_COOKIE` | empty | Optional raw Bilibili `Cookie` header value. Used by the Bilibili API resolver, direct MP4 downloader, ffmpeg fallback, and yt-dlp header fallback. Keep it private. |
| `BILIBILI_QUALITY` | `80` | Preferred Bilibili API quality. `80` is 1080P, `64` is 720P. |
| `BILIBILI_QUALITY_FALLBACKS` | `80,64,32,16` | Bilibili API fallback quality list. The service tries each value until a usable stream is found. |
| `FORMAT_SELECTOR` | `bv*[vcodec^=avc1]+ba[ext=m4a]/b[vcodec^=avc1]/bv*[vcodec^=avc1]+ba/bv*+ba/b` | yt-dlp format selector. Defaults to H.264 and m4a-first output for better VRChat/MP4 compatibility. |
| `FFMPEG_PATH` | `ffmpeg` | Path to ffmpeg. |
| `MAX_CONCURRENT_JOBS` | `1` | Concurrent background download/upload workers. HTTP requests return immediately after a job is queued. |
| `JOB_QUEUE_SIZE` | `16` | Maximum queued jobs waiting for a worker. When full, new job requests return `503` quickly. |
| `JOB_TIMEOUT_MINUTES` | `90` | Per command timeout. |
| `DIRECT_PLAYBACK_MODE` | `proxy` | Behavior for `/?v=BVxxxx`. Use `proxy` to stream through this service, or `redirect` to return a 302 temporary MP4 link. |
| `ALLOWED_HOSTS` | `bilibili.com,www.bilibili.com,m.bilibili.com,b23.tv,youtube.com,www.youtube.com,m.youtube.com,music.youtube.com,youtu.be` | Comma-separated allowed source hosts. |
| `R2_ENDPOINT` | empty | Cloudflare R2 S3 endpoint, usually `https://<account-id>.r2.cloudflarestorage.com`. |
| `R2_ACCESS_KEY_ID` | empty | R2 API token access key ID. |
| `R2_SECRET_ACCESS_KEY` | empty | R2 API token secret access key. |
| `R2_BUCKET` | empty | R2 bucket name. |
| `R2_PUBLIC_BASE_URL` | empty | Public base URL for the bucket or custom domain, such as `https://video.example.com`. |
| `R2_KEY_PREFIX` | `vrchat` | Object key prefix used before `BVxxxx/mp4/video.mp4`, `youtube-xxxxxxxxxxx/mp4/video.mp4`, or HLS equivalents. |
| `R2_CACHE_CONTROL` | `public, max-age=86400` | Cache-Control metadata applied to uploaded objects. |
| `R2_UPLOAD_TIMEOUT_SECONDS` | `600` | Timeout for each R2 upload request. |

## Cloudflare R2 Mode

Create an R2 bucket, bind a public/custom domain to it, then create an R2 API token with object read/write permission for that bucket.

Example:

```bash
export R2_ENDPOINT="https://<account-id>.r2.cloudflarestorage.com"
export R2_ACCESS_KEY_ID="<access-key-id>"
export R2_SECRET_ACCESS_KEY="<secret-access-key>"
export R2_BUCKET="vrchat-video"
export R2_PUBLIC_BASE_URL="https://video.example.com"
export R2_KEY_PREFIX="vrchat"

go run ./cmd/server
```

A submitted MP4 job will upload:

```text
vrchat/BVxxxx/mp4/video.mp4
vrchat/youtube-xxxxxxxxxxx/mp4/video.mp4
```

A submitted HLS job will upload:

```text
vrchat/BVxxxx/hls/index.m3u8
vrchat/BVxxxx/hls/segment_00000.ts
...
```

The API returns the public URL to `video.mp4` or `index.m3u8`.

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

- Bilibili can return HTTP 412 or require cookies even for public videos. For the server-side resolver, set `BILIBILI_COOKIE="SESSDATA=...; bili_jct=...; DedeUserID=..."` in `.env`. For yt-dlp-only workflows, you can also export a Netscape-format cookies file and set `YTDLP_COOKIES_FILE=/path/to/cookies.txt`, or try `YTDLP_COOKIES_FROM_BROWSER=chrome`, `edge`, or `firefox`; browser-profile reads can fail while the browser profile is locked.
- Bilibili 1080P usually requires a valid login cookie. The default API quality preference is 1080P, then 720P and lower fallbacks when higher quality is unavailable.
- YouTube support uses `yt-dlp`, so region restrictions, age checks, sign-in checks, or rate limits may still require cookies or a different network environment.
- HTTP conversion jobs require R2 and automatically remove local generated media after the upload attempt finishes.
- Only process videos you have the right to play or share.
