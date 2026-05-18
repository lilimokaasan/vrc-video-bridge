package streamer

const indexHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>KoiMoe VRChat Video</title>
  <style>
    :root {
      --pink: #fb98c0;
      --pink-deep: #f071a8;
      --petal: #ffeeeb;
      --warm: #fe9600;
      --ink: #4d4650;
      --muted: #8d7e8a;
      --paper: rgba(255, 255, 255, .76);
      --paper-strong: rgba(255, 255, 255, .9);
      --line: rgba(251, 152, 192, .28);
      --shadow: 0 22px 60px rgba(218, 116, 157, .22);
    }

    * { box-sizing: border-box; }

    html {
      min-height: 100%;
      scrollbar-color: var(--pink) #ffeeeb;
      scrollbar-width: thin;
      scroll-behavior: smooth;
    }

    body {
      min-height: 100vh;
      margin: 0;
      color: var(--ink);
      font-family: "Segoe UI", "Noto Sans SC", "Microsoft YaHei", system-ui, sans-serif;
      background:
        linear-gradient(180deg, rgba(255,255,255,.28), rgba(255,238,245,.82)),
        url("/assets/sakura-branch-pastel.jpg") center / cover fixed;
      letter-spacing: 0;
    }

    body::before {
      content: "";
      position: fixed;
      inset: 0;
      pointer-events: none;
      background:
        linear-gradient(120deg, rgba(255,255,255,.68), rgba(255,246,250,.42) 48%, rgba(255,255,255,.78)),
        repeating-linear-gradient(90deg, rgba(255,255,255,.16) 0 1px, transparent 1px 64px);
      z-index: -1;
    }

    ::-webkit-scrollbar { width: 6px; height: 6px; }
    ::-webkit-scrollbar-track { background: #ffeeeb; }
    ::-webkit-scrollbar-thumb { background: var(--pink); border-radius: 25px; }
    ::-webkit-scrollbar-thumb:hover { background: var(--pink-deep); }

    .scrollbar-progress {
      position: fixed;
      top: 0;
      left: 0;
      width: 100%;
      height: 3px;
      z-index: 20;
      transform: scaleX(0);
      transform-origin: left center;
      background: linear-gradient(90deg, var(--pink), #ffd2e3);
      box-shadow: 0 0 12px rgba(251, 152, 192, .7);
    }

    .shell {
      width: min(1120px, calc(100% - 32px));
      min-height: 100vh;
      margin: 0 auto;
      display: grid;
      grid-template-rows: auto 1fr auto;
      gap: 28px;
      padding: 24px 0 28px;
    }

    header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 18px;
      color: rgba(77, 70, 80, .78);
    }

    .brand {
      display: flex;
      align-items: center;
      gap: 12px;
      min-width: 0;
    }

    .mark {
      width: 42px;
      height: 42px;
      display: grid;
      place-items: center;
      border-radius: 50%;
      background: rgba(255,255,255,.78);
      border: 1px solid rgba(255,255,255,.9);
      box-shadow: 0 12px 34px rgba(251,152,192,.22);
      color: var(--pink-deep);
      font-size: 20px;
    }

    .brand-title {
      margin: 0;
      font-size: 18px;
      font-weight: 600;
      color: #4b414c;
    }

    .brand-subtitle {
      margin: 2px 0 0;
      color: var(--muted);
      font-size: 12px;
    }

    .status-pill {
      flex: 0 0 auto;
      padding: 8px 13px;
      border-radius: 999px;
      border: 1px solid rgba(251,152,192,.3);
      background: rgba(255,255,255,.62);
      color: #8a5a70;
      font-size: 12px;
      backdrop-filter: blur(14px);
    }

    main {
      display: grid;
      grid-template-columns: minmax(0, 1.1fr) 340px;
      align-items: center;
      gap: 26px;
    }

    .panel {
      border: 1px solid rgba(255,255,255,.82);
      background: var(--paper);
      box-shadow: var(--shadow);
      backdrop-filter: blur(18px);
    }

    .converter {
      border-radius: 18px;
      padding: clamp(24px, 4vw, 46px);
    }

    .eyebrow {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      margin: 0 0 14px;
      padding: 7px 12px;
      border-radius: 999px;
      color: #b76083;
      background: rgba(255,238,235,.72);
      border: 1px solid rgba(251,152,192,.2);
      font-size: 12px;
    }

    h1 {
      margin: 0;
      max-width: 720px;
      font-size: clamp(34px, 5vw, 62px);
      line-height: 1.06;
      font-weight: 700;
      color: #443a44;
    }

    .lead {
      max-width: 620px;
      margin: 18px 0 28px;
      color: #7e6f7b;
      font-size: 16px;
      line-height: 1.9;
    }

    form {
      display: grid;
      gap: 14px;
    }

    .input-wrap {
      display: grid;
      grid-template-columns: 1fr auto;
      gap: 10px;
      align-items: center;
      padding: 10px;
      border: 1px solid var(--line);
      border-radius: 15px;
      background: rgba(255,255,255,.72);
      box-shadow: inset 0 1px 0 rgba(255,255,255,.9);
    }

    input {
      width: 100%;
      min-width: 0;
      border: 0;
      outline: 0;
      background: transparent;
      padding: 12px 10px;
      color: var(--ink);
      font: inherit;
      font-size: 15px;
    }

    input::placeholder { color: rgba(141,126,138,.68); }

    .input-wrap:focus-within {
      border-color: var(--pink);
      box-shadow: 0 0 0 4px rgba(251,152,192,.15), inset 0 1px 0 rgba(255,255,255,.9);
    }

    button, .copy-link {
      border: 0;
      cursor: pointer;
      font: inherit;
      transition: transform .18s ease, box-shadow .18s ease, background .18s ease;
    }

    button:disabled {
      cursor: wait;
      opacity: .68;
    }

    .submit {
      min-height: 46px;
      padding: 0 20px;
      border-radius: 12px;
      color: white;
      background: linear-gradient(135deg, var(--pink), #ffabcf);
      box-shadow: 0 14px 28px rgba(251,152,192,.34);
      white-space: nowrap;
    }

    .submit:hover:not(:disabled) { transform: translateY(-1px); box-shadow: 0 18px 34px rgba(251,152,192,.42); }

    .options {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      align-items: center;
    }

    .segmented {
      display: inline-grid;
      grid-template-columns: 1fr 1fr;
      gap: 4px;
      padding: 4px;
      border-radius: 999px;
      border: 1px solid rgba(251,152,192,.22);
      background: rgba(255,255,255,.62);
    }

    .segmented label {
      position: relative;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      min-width: 88px;
      min-height: 34px;
      padding: 0 14px;
      border-radius: 999px;
      color: #967284;
      font-size: 13px;
      cursor: pointer;
    }

    .segmented input {
      position: absolute;
      inset: 0;
      opacity: 0;
      cursor: pointer;
    }

    .segmented label:has(input:checked) {
      color: white;
      background: var(--pink);
      box-shadow: 0 8px 18px rgba(251,152,192,.28);
    }

    .hint {
      color: #9a8794;
      font-size: 12px;
    }

    .result {
      display: none;
      margin-top: 22px;
      padding: 16px;
      border-radius: 15px;
      border: 1px solid rgba(251,152,192,.22);
      background: rgba(255,255,255,.68);
    }

    .result.is-visible { display: block; }

    .result-label {
      margin: 0 0 8px;
      color: #b76083;
      font-size: 12px;
    }

    .result-row {
      display: grid;
      grid-template-columns: 1fr auto;
      gap: 10px;
      align-items: center;
    }

    .result a {
      min-width: 0;
      overflow-wrap: anywhere;
      color: #d95f95;
      text-decoration: none;
      line-height: 1.5;
    }

    .copy-link {
      min-height: 38px;
      padding: 0 13px;
      border-radius: 10px;
      color: #b76083;
      background: #ffeeeb;
    }

    .side {
      display: grid;
      gap: 14px;
    }

    .note {
      border-radius: 14px;
      padding: 18px;
    }

    .note h2 {
      margin: 0 0 10px;
      font-size: 16px;
      color: #5b4a58;
    }

    .note p {
      margin: 0;
      color: #857684;
      line-height: 1.75;
      font-size: 13px;
    }

    .steps {
      display: grid;
      gap: 9px;
      margin-top: 14px;
    }

    .step {
      display: grid;
      grid-template-columns: 26px 1fr;
      gap: 10px;
      align-items: start;
      color: #7e6f7b;
      font-size: 13px;
    }

    .dot {
      width: 26px;
      height: 26px;
      display: grid;
      place-items: center;
      border-radius: 50%;
      color: white;
      background: var(--pink);
      font-size: 12px;
      box-shadow: 0 8px 18px rgba(251,152,192,.28);
    }

    .log {
      min-height: 120px;
      max-height: 240px;
      overflow: auto;
      margin: 0;
      padding: 14px;
      border-radius: 12px;
      color: #776370;
      background: rgba(255,255,255,.66);
      border: 1px solid rgba(251,152,192,.18);
      font-size: 12px;
      line-height: 1.6;
      white-space: pre-wrap;
    }

    footer {
      color: rgba(126,111,123,.76);
      font-size: 12px;
      text-align: center;
    }

    @media (max-width: 880px) {
      .shell { width: min(100% - 22px, 680px); padding-top: 16px; }
      header { align-items: flex-start; }
      main { grid-template-columns: 1fr; align-items: stretch; }
      .input-wrap { grid-template-columns: 1fr; }
      .submit { width: 100%; }
      .status-pill { display: none; }
    }

    @media (prefers-reduced-motion: reduce) {
      html { scroll-behavior: auto; }
      *, *::before, *::after { transition-duration: .01ms !important; animation-duration: .01ms !important; }
    }
  </style>
</head>
<body>
  <div class="scrollbar-progress" aria-hidden="true"></div>
  <div class="shell">
    <header>
      <div class="brand">
        <div class="mark" aria-hidden="true">桜</div>
        <div>
          <p class="brand-title">KoiMoe VRChat Video</p>
          <p class="brand-subtitle">恋と萌えの小さな変換室</p>
        </div>
      </div>
      <div class="status-pill" id="r2Hint">R2 public links when configured</div>
    </header>

    <main>
      <section class="converter panel" aria-labelledby="title">
        <p class="eyebrow">Bilibili → VRChat Player</p>
        <h1 id="title">把一段视频，轻轻递到 VRChat 里。</h1>
        <p class="lead">贴入 Bilibili 视频地址，服务会在后台生成 VRChat 播放器可用的链接。配置 R2 后，完成时会返回公开对象存储地址。</p>

        <form id="convertForm">
          <div class="input-wrap">
            <input id="url" name="url" autocomplete="off" required placeholder="https://www.bilibili.com/video/BV..." />
            <button class="submit" id="submit" type="submit">开始转换</button>
          </div>

          <div class="options" aria-label="输出格式">
            <div class="segmented">
              <label><input type="radio" name="format" value="hls" checked />HLS</label>
              <label><input type="radio" name="format" value="mp4" />MP4</label>
            </div>
            <span class="hint">HLS 更适合尝试 VRChat；MP4 文件更简单。</span>
          </div>
        </form>

        <div class="result" id="result">
          <p class="result-label">可播放链接</p>
          <div class="result-row">
            <a id="playback" href="#" target="_blank" rel="noreferrer"></a>
            <button class="copy-link" id="copy" type="button">复制</button>
          </div>
        </div>
      </section>

      <aside class="side">
        <section class="note panel">
          <h2>小小流程</h2>
          <p>页面会创建任务并等待它完成。你可以把最终链接贴进 VRChat 视频播放器。</p>
          <div class="steps">
            <div class="step"><span class="dot">1</span><span>解析 Bilibili 视频</span></div>
            <div class="step"><span class="dot">2</span><span>生成 HLS 或 MP4</span></div>
            <div class="step"><span class="dot">3</span><span>上传 R2 或使用本地链接</span></div>
          </div>
        </section>

        <section class="note panel">
          <h2>任务状态</h2>
          <pre class="log" id="log">等待一条 Bilibili 链接...</pre>
        </section>
      </aside>
    </main>

    <footer>Soft links for tiny rooms and shared songs.</footer>
  </div>

  <script>
    const form = document.querySelector('#convertForm');
    const submit = document.querySelector('#submit');
    const log = document.querySelector('#log');
    const result = document.querySelector('#result');
    const playback = document.querySelector('#playback');
    const copy = document.querySelector('#copy');
    const progress = document.querySelector('.scrollbar-progress');

    const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));

    function setLog(message, detail) {
      const suffix = detail ? '\n\n' + JSON.stringify(detail, null, 2) : '';
      log.textContent = message + suffix;
    }

    function updateProgress() {
      const max = document.documentElement.scrollHeight - window.innerHeight;
      progress.style.transform = 'scaleX(' + (max > 0 ? window.scrollY / max : 0) + ')';
    }

    async function createJob(url, format) {
      const res = await fetch('/api/jobs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url, format })
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || '任务创建失败');
      return data;
    }

    async function poll(statusURL) {
      for (;;) {
        const res = await fetch(statusURL);
        const job = await res.json();
        if (!res.ok) throw new Error(job.error || '任务查询失败');

        if (job.status === 'ready') {
          setLog('转换完成。链接已经准备好啦。', job);
          playback.href = job.playback_url;
          playback.textContent = job.playback_url;
          result.classList.add('is-visible');
          return;
        }
        if (job.status === 'failed') {
          setLog('转换失败。', job);
          throw new Error(job.error || '转换失败');
        }

        setLog(job.message || '正在转换中，请稍等...', job);
        await sleep(2500);
      }
    }

    form.addEventListener('submit', async event => {
      event.preventDefault();
      result.classList.remove('is-visible');
      submit.disabled = true;
      submit.textContent = '转换中...';

      const url = document.querySelector('#url').value.trim();
      const format = new FormData(form).get('format');

      try {
        setLog('正在创建任务...');
        const job = await createJob(url, format);
        setLog('任务已经创建，开始等待生成链接...', job);
        await poll(job.status_url.replace(location.origin, ''));
      } catch (error) {
        setLog(error.message);
      } finally {
        submit.disabled = false;
        submit.textContent = '开始转换';
      }
    });

    copy.addEventListener('click', async () => {
      if (!playback.textContent) return;
      await navigator.clipboard.writeText(playback.textContent);
      copy.textContent = '已复制';
      setTimeout(() => copy.textContent = '复制', 1200);
    });

    window.addEventListener('scroll', updateProgress, { passive: true });
    window.addEventListener('resize', updateProgress);
    updateProgress();
  </script>
</body>
</html>
`
