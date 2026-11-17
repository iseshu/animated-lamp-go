# 🚀 Animated Lamp (Go) - Ultra-Fast In-Memory Telegram Screenshot Bot

A high-performance Go rewrite of [**`odysseusmax/animated-lamp`**](https://github.com/odysseusmax/animated-lamp) by [@odysseusmax](https://github.com/odysseusmax).

Engineered specifically for **ultra-low resource hosting (512MB RAM & 512MB Disk)** using **100% in-memory streaming pipelines** (zero disk writes for screenshots) and an embedded **bbolt** database.

---

## ⚡ Native MTProto Client + Embedded Local Streamer

Unlike the original Python bot which required a **separate, external, closed-source streaming server** (`HOST`), this Go version features an **all-in-one architecture**:
* Uses **`gotd/td`** (pure Go MTProto framework) with your **`API_ID`** and **`API_HASH`**.
* Spins up an **embedded HTTP Range streamer on `127.0.0.1`** inside the exact same binary.
* Directly fetches byte ranges on demand from Telegram DC servers and pipes them into FFmpeg in RAM.
* **No 20MB file limit**—handles **2 GB** (and 4 GB Premium) video files sent directly inside Telegram chats!
* If `API_ID` / `API_HASH` are not provided, it gracefully falls back to Telegram Bot API mode.

---

## 🙏 Credits & Acknowledgements

* **Original Concept & Bot**: [@odysseusmax](https://github.com/odysseusmax) for creating the original [**`animated-lamp`**](https://github.com/odysseusmax/animated-lamp) (`@screenshotit_bot`).
* **Go Implementation**: Rewritten in Go to eliminate disk bottlenecks, slash RAM usage from ~150MB down to ~25MB, and embed native MTProto streaming without any external proxy servers.

---

## ⚡ Key Improvements Over Original Python Bot

| Feature | Original Python (`animated-lamp`) | Go Rewrite (`animated-lamp-go`) |
| :--- | :--- | :--- |
| **Telegram Protocol** | Pyrogram MTProto | **Native Go MTProto (`gotd/td`)** + Bot API fallback |
| **Streaming Mechanism** | Requires **external closed-source server** (`HOST`) | **Embedded in-memory HTTP Range streamer (`127.0.0.1`)** |
| **Pipeline Storage** | Writes intermediate `.png` files to disk | **100% In-Memory (`pipe:1`)**; 0 bytes written to disk |
| **Idle Memory (RAM)** | ~150 MB – 180 MB | **~25 MB** |
| **Compiled Binary** | Full Python environment + wheels (~300MB) | **Single static binary (~18 MB)** |
| **Database** | External MongoDB cluster required | **Embedded bbolt** (zero external dependencies) |
| **Container Size** | ~450 MB – 600 MB | **< 80 MB (Alpine + FFmpeg)** |
| **Concurrency Cap** | Unthrottled (default 20, prone to OOM) | **Semaphore-throttled worker pool** (safe for 512MB RAM) |
| **Filter Security** | Unescaped filter strings | **Strictly sanitized drawtext filter arguments** |

---

## 📋 Features

1. **Native Telegram Videos & Direct URLs**:
   - Stream native Telegram videos up to **2 GB** using MTProto.
   - Stream direct HTTP/HTTPS video links (MP4, MKV, WebM, HLS/M3U8).
2. **Batch Screenshots (2–10 frames)**:
   - Configurable equal interval or random timestamp distribution.
   - Piped in-memory directly to Telegram photo/document media groups.
3. **Manual Timestamps**:
   - Custom timestamps via comma-separated seconds or `HH:MM:SS`.
4. **Media Information Inspector (`/mediainfo`)**:
   - Streams format, codec, bitrate, audio, and subtitle streams via `ffprobe`.
5. **Video Sample & Trimming**:
   - Fast stream-copy (`-c copy`) trimming and preview generation.
6. **Customizable Watermarking**:
   - Custom text up to 30 characters.
   - 12 colors & 3 dynamic font sizes (scaled with video resolution).
   - 9 positioning anchors.
7. **Delivery Modes**:
   - Photo mode (compressed, instant gallery preview).
   - Document mode (uncompressed, full quality).
8. **Admin Suite**:
   - `/status`: Live RAM allocations, active workers, uptime, and user counts.

---

## 🛠️ Environment Variables

| Variable | Required | Default | Description |
| :--- | :---: | :---: | :--- |
| `BOT_TOKEN` | **Yes** | — | Telegram Bot token from @BotFather |
| `API_ID` | Recommended | `0` | App ID from [my.telegram.org](https://my.telegram.org) (enables native 2GB MTProto streaming) |
| `API_HASH` | Recommended | `""` | App Hash from [my.telegram.org](https://my.telegram.org) |
| `AUTH_USERS` | No | — | Space-separated list of Admin Telegram IDs |
| `DB_PATH` | No | `data/bot.db` | Path to the bbolt embedded database file |
| `MAX_WORKERS` | No | `2` | Max concurrent FFmpeg operations (keep `2` for 512MB RAM) |
| `MAX_PROCESSES_PER_USER` | No | `1` | Max concurrent operations per user |
| `SLOW_SPEED_DELAY` | No | `5` | Anti-spam cooldown delay in seconds |
| `TIMEOUT` | No | `300` | Max duration per operation in seconds |

---

## 🐳 Docker Deployment

```bash
docker build -t animated-lamp-go .
docker run -d \
  --name animated-lamp-go \
  -e BOT_TOKEN="your_bot_token" \
  -e API_ID="123456" \
  -e API_HASH="abcdef123456..." \
  -e AUTH_USERS="12345678" \
  -v $(pwd)/data:/app/data \
  animated-lamp-go
```

---

## 💻 Manual Build & Run

Ensure `ffmpeg` and `ffprobe` are installed on your host system:

```bash
# Build static binary
CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/screenshotbot ./cmd/bot

# Run in MTProto mode
export BOT_TOKEN="your_bot_token"
export API_ID="your_api_id"
export API_HASH="your_api_hash"
./bin/screenshotbot
```

---

## 📄 License

GPL-3.0 (following the original repository license by @odysseusmax).
