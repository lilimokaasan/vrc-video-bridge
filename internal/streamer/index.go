package streamer

const indexHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Bilibili to VRChat Streamer</title>
  <style>
    body { font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; max-width: 760px; margin: 48px auto; padding: 0 20px; line-height: 1.6; color: #20242a; }
    input, select, button { font: inherit; padding: 10px 12px; border: 1px solid #ccd2dd; border-radius: 8px; }
    input { width: min(100%, 560px); }
    button { background: #fb98c0; color: white; border-color: #fb98c0; cursor: pointer; }
    pre { background: #f6f7f9; padding: 16px; border-radius: 8px; white-space: pre-wrap; }
    .row { display: flex; flex-wrap: wrap; gap: 10px; margin: 16px 0; }
  </style>
</head>
<body>
  <h1>Bilibili to VRChat Streamer</h1>
  <p>输入 Bilibili 视频链接，服务会用 yt-dlp 下载并用 ffmpeg 输出 HLS 或 MP4。VRChat 通常优先试 HLS 的 <code>.m3u8</code> 播放地址。</p>
  <div class="row">
    <input id="url" placeholder="https://www.bilibili.com/video/BV..." />
    <select id="format">
      <option value="hls">HLS .m3u8</option>
      <option value="mp4">MP4</option>
    </select>
    <button id="submit">创建任务</button>
  </div>
  <pre id="output">等待输入...</pre>
  <script>
    const output = document.querySelector('#output');
    const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));
    async function poll(statusURL) {
      for (;;) {
        const res = await fetch(statusURL);
        const job = await res.json();
        output.textContent = JSON.stringify(job, null, 2);
        if (job.status === 'ready' || job.status === 'failed') return;
        await sleep(2500);
      }
    }
    document.querySelector('#submit').addEventListener('click', async () => {
      output.textContent = '创建任务中...';
      const res = await fetch('/api/jobs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          url: document.querySelector('#url').value,
          format: document.querySelector('#format').value
        })
      });
      const data = await res.json();
      output.textContent = JSON.stringify(data, null, 2);
      if (data.status_url) poll(data.status_url.replace(location.origin, ''));
    });
  </script>
</body>
</html>
`
