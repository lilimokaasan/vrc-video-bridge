package streamer

const indexHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="icon" href="/favicon.png?v=20260518-transparent" type="image/png">
  <link rel="shortcut icon" href="/favicon.ico?v=20260518-transparent">
  <title>KoiMoe Link Room</title>
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
      scrollbar-color: var(--pink) rgba(255,238,235,.72);
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

    body::after {
      content: "";
      position: fixed;
      inset: auto 0 0;
      height: 34vh;
      pointer-events: none;
      background: linear-gradient(180deg, transparent, rgba(255,238,245,.64));
      z-index: -1;
    }

    ::-webkit-scrollbar { width: 6px; height: 6px; }
    ::-webkit-scrollbar-track { background: rgba(255,238,235,.72); }
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
      transition: transform 1s cubic-bezier(.22, .9, .28, 1), background 1s ease;
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
      position: relative;
      overflow: hidden;
      border: 1px solid rgba(255,255,255,.82);
      background: var(--paper);
      box-shadow: var(--shadow);
      backdrop-filter: blur(18px);
    }

    .panel::before {
      content: "";
      position: absolute;
      inset: 0 0 auto;
      height: 1px;
      background: linear-gradient(90deg, transparent, rgba(255,255,255,.95), transparent);
      pointer-events: none;
    }

    .converter {
      border-radius: 18px;
      padding: clamp(24px, 4vw, 46px);
      background:
        linear-gradient(150deg, rgba(255,255,255,.88), rgba(255,247,251,.74) 52%, rgba(255,255,255,.78)),
        var(--paper);
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
      box-shadow: 0 10px 28px rgba(251,152,192,.1), inset 0 1px 0 rgba(255,255,255,.9);
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
      position: relative;
      display: inline-grid;
      grid-template-columns: 1fr 1fr;
      padding: 4px;
      border-radius: 999px;
      border: 1px solid rgba(251,152,192,.22);
      background: rgba(255,255,255,.72);
      box-shadow: inset 0 1px 0 rgba(255,255,255,.95), 0 10px 22px rgba(251,152,192,.12);
    }

    .segmented-thumb {
      position: absolute;
      top: 4px;
      left: 4px;
      width: calc((100% - 8px) / 2);
      height: calc(100% - 8px);
      border-radius: 999px;
      background: linear-gradient(135deg, var(--pink), #ffb1d2);
      box-shadow: 0 8px 18px rgba(251,152,192,.32);
      transform: translateX(0);
      transition: transform .34s cubic-bezier(.22, .9, .28, 1), box-shadow .24s ease;
      pointer-events: none;
    }

    .segmented[data-format="mp4"] .segmented-thumb {
      transform: translateX(100%);
    }

    .segmented label {
      position: relative;
      z-index: 1;
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
      transition: color .24s ease, transform .24s ease;
    }

    .segmented input {
      position: absolute;
      inset: 0;
      opacity: 0;
      cursor: pointer;
    }

    .segmented label:has(input:checked) {
      color: white;
      transform: translateY(-.5px);
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
      background: linear-gradient(180deg, rgba(255,255,255,.78), rgba(255,247,251,.66));
      box-shadow: inset 0 1px 0 rgba(255,255,255,.88), 0 12px 30px rgba(251,152,192,.1);
    }

    .result.is-visible { display: block; }

    .result-label {
      margin: 0 0 8px;
      color: #b76083;
      font-size: 12px;
    }

    .result-label:not(:first-child) {
      margin-top: 14px;
    }

    .result-row {
      display: grid;
      grid-template-columns: 1fr auto;
      gap: 10px;
      align-items: center;
      min-height: 44px;
      padding: 6px 6px 6px 12px;
      border-radius: 12px;
      border: 1px solid rgba(251,152,192,.14);
      background: rgba(255,255,255,.58);
    }

    .result a {
      min-width: 0;
      color: #d95f95;
      text-decoration: none;
      line-height: 1.5;
      cursor: pointer;
    }

    .result a.is-collapsed {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .result a.is-expanded {
      overflow-wrap: anywhere;
      white-space: normal;
    }

    .copy-link {
      width: 76px;
      flex: 0 0 76px;
      min-height: 38px;
      padding: 0 13px;
      border-radius: 10px;
      color: #b76083;
      background: linear-gradient(180deg, #fff4f6, #ffeeeb);
      box-shadow: inset 0 1px 0 rgba(255,255,255,.82);
      text-align: center;
      white-space: nowrap;
    }

    .copy-link:hover { transform: translateY(-1px); box-shadow: 0 8px 16px rgba(251,152,192,.16); }

    .job-progress {
      display: none;
      gap: 10px;
      margin-top: 14px;
      padding: 14px;
      border-radius: 14px;
      border: 1px solid rgba(251,152,192,.18);
      background: rgba(255,255,255,.58);
    }

    .job-progress.is-visible {
      display: grid;
    }

    .progress-item {
      display: grid;
      gap: 6px;
    }

    .progress-meta {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      color: #8a7180;
      font-size: 12px;
    }

    .progress-track {
      height: 7px;
      overflow: hidden;
      border-radius: 999px;
      background: rgba(251,152,192,.14);
      box-shadow: inset 0 1px 2px rgba(141,126,138,.1);
    }

    .progress-fill {
      width: 100%;
      height: 100%;
      border-radius: inherit;
      background: linear-gradient(90deg, var(--pink), #ffd1e3);
      transform: scaleX(0);
      transform-origin: left center;
      transition: transform .38s ease;
    }

    .progress-item.is-active .progress-fill.is-indeterminate {
      transform: scaleX(.42);
      animation: progress-drift 1.4s ease-in-out infinite;
    }

    .progress-item.is-done .progress-fill {
      transform: scaleX(1);
    }

    @keyframes progress-drift {
      0% { transform: translateX(-92%) scaleX(.42); }
      50% { transform: translateX(58%) scaleX(.42); }
      100% { transform: translateX(238%) scaleX(.42); }
    }

    .side {
      display: grid;
      gap: 14px;
    }

    .note {
      border-radius: 14px;
      padding: 18px;
      background: linear-gradient(180deg, rgba(255,255,255,.76), rgba(255,248,251,.62));
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
          <p class="brand-title">KoiMoe Link Room</p>
          <p class="brand-subtitle">恋と萌えの小さな場所</p>
        </div>
      </div>
      <div class="status-pill" id="r2Hint">把喜欢的片段轻轻收好</div>
    </header>

    <main>
      <section class="converter panel" aria-labelledby="title">
        <p class="eyebrow">For VRChat watch moments</p>
        <h1 id="title">把喜欢的视频，轻轻递到 VRChat 里。</h1>
        <p class="lead">贴入一条 Bilibili 地址，稍等片刻，就会整理出适合在 VRChat 里和朋友一起看的链接。</p>

        <form id="convertForm">
          <div class="input-wrap">
            <input id="url" name="url" autocomplete="off" required placeholder="https://www.bilibili.com/video/BV..." />
            <button class="submit" id="submit" type="submit">轻轻整理</button>
          </div>

          <div class="options" aria-label="输出格式">
            <div class="segmented" data-format="mp4">
              <span class="segmented-thumb" aria-hidden="true"></span>
              <label><input type="radio" name="format" value="hls" />HLS</label>
              <label><input type="radio" name="format" value="mp4" checked />MP4</label>
            </div>
            <span class="hint">MP4 通常更适合 VRChat 播放；HLS 可以作为另一种尝试。</span>
          </div>
        </form>

        <div class="result" id="result">
          <p class="result-label">先到的小纸条</p>
          <div class="result-row">
            <a id="direct" href="#" target="_blank" rel="noreferrer"></a>
            <button class="copy-link" data-copy-target="direct" type="button">复制</button>
          </div>

          <p class="result-label">整理好的分享链接</p>
          <div class="result-row">
            <a id="playback" href="#" target="_blank" rel="noreferrer"></a>
            <button class="copy-link" data-copy-target="playback" type="button">复制</button>
          </div>
        </div>

        <div class="job-progress" id="jobProgress" aria-live="polite">
          <div class="progress-item" data-progress-key="download">
            <div class="progress-meta">
              <span class="progress-name">下载 MP4</span>
              <span class="progress-value">等待中</span>
            </div>
            <div class="progress-track"><span class="progress-fill"></span></div>
          </div>
          <div class="progress-item" data-progress-key="upload">
            <div class="progress-meta">
              <span class="progress-name">上传到 R2</span>
              <span class="progress-value">等待中</span>
            </div>
            <div class="progress-track"><span class="progress-fill"></span></div>
          </div>
        </div>
      </section>

      <aside class="side">
        <section class="note panel">
          <h2>小小流程</h2>
          <p>页面会先找到视频的临时入口，再慢慢整理成适合 VRChat 播放的分享链接。</p>
          <div class="steps">
            <div class="step"><span class="dot">1</span><span>找到视频入口</span></div>
            <div class="step"><span class="dot">2</span><span>整理成可播放的片段</span></div>
            <div class="step"><span class="dot">3</span><span>送出一条分享链接</span></div>
          </div>
        </section>

        <section class="note panel">
          <h2>小小回声</h2>
          <pre class="log" id="log">等一条想分享的视频链接...</pre>
        </section>
      </aside>
    </main>

    <footer>A soft diary for tiny heartbeats, favorite things, and VRChat moments.</footer>
  </div>

  <script>
    if (!window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      document.write('<script src="/assets/smooth-scroll.js"><\/script>');
    }
  </script>
  <script>
    const form = document.querySelector('#convertForm');
    const submit = document.querySelector('#submit');
    const log = document.querySelector('#log');
    const result = document.querySelector('#result');
    const direct = document.querySelector('#direct');
    const playback = document.querySelector('#playback');
    const jobProgress = document.querySelector('#jobProgress');
    const progressItems = document.querySelectorAll('[data-progress-key]');
    const copyButtons = document.querySelectorAll('[data-copy-target]');
    const segmented = document.querySelector('.segmented');
    const progress = document.querySelector('.scrollbar-progress');

    const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));

    function setLog(message) {
      log.textContent = message;
    }

    function updateProgress() {
      const max = document.documentElement.scrollHeight - window.innerHeight;
      progress.style.transform = 'scaleX(' + (max > 0 ? window.scrollY / max : 0) + ')';
    }

    function syncFormatControl() {
      const selected = new FormData(form).get('format') || 'mp4';
      segmented.dataset.format = selected;
    }

    function compactURL(url) {
      if (!url || url.length <= 72) return url;
      return url.slice(0, 42) + '...' + url.slice(-24);
    }

    function setLink(anchor, url, waitingText) {
      const previousURL = anchor.dataset.fullUrl || '';
      if (url) {
        anchor.href = url;
        anchor.dataset.fullUrl = url;
        if (previousURL !== url) {
          anchor.dataset.expanded = 'false';
        }
        const expanded = anchor.dataset.expanded === 'true';
        anchor.textContent = expanded ? url : compactURL(url);
        anchor.title = expanded ? '点击收起链接' : '点击展开完整链接';
        anchor.classList.toggle('is-expanded', expanded);
        anchor.classList.toggle('is-collapsed', !expanded);
      } else {
        anchor.removeAttribute('href');
        anchor.dataset.fullUrl = '';
        anchor.dataset.expanded = 'false';
        anchor.textContent = waitingText;
        anchor.title = '';
        anchor.classList.remove('is-expanded', 'is-collapsed');
      }
    }

    function renderLinks(job) {
      setLink(direct, job.direct_url, '正在寻找视频入口...');
      setLink(playback, job.playback_url, '正在整理分享链接...');
      if (job.direct_url || job.playback_url) {
        result.classList.add('is-visible');
      }
    }

    function resetJobProgress() {
      jobProgress.classList.remove('is-visible');
      progressItems.forEach(item => {
        item.classList.remove('is-active', 'is-done');
        item.querySelector('.progress-value').textContent = '等待中';
        const fill = item.querySelector('.progress-fill');
        fill.classList.remove('is-indeterminate');
        fill.style.transform = 'scaleX(0)';
      });
    }

    function renderProgress(job) {
      const progressMap = job.progress || {};
      const hasProgress = Object.keys(progressMap).length > 0;
      jobProgress.classList.toggle('is-visible', hasProgress);
      progressItems.forEach(item => {
        const key = item.dataset.progressKey;
        const step = progressMap[key];
        const value = item.querySelector('.progress-value');
        const fill = item.querySelector('.progress-fill');
        item.classList.remove('is-active', 'is-done');
        fill.classList.remove('is-indeterminate');
        if (!step) {
          value.textContent = '等待中';
          fill.style.transform = 'scaleX(0)';
          return;
        }
        if (step.state === 'done') {
          item.classList.add('is-done');
          value.textContent = '完成';
          fill.style.transform = 'scaleX(1)';
          return;
        }
        item.classList.add('is-active');
        if (step.bytes_total > 0 || step.percent > 0) {
          const percent = Math.max(0, Math.min(100, step.percent || 0));
          value.textContent = percent + '%';
          fill.style.transform = 'scaleX(' + (percent / 100) + ')';
        } else {
          value.textContent = step.message || '进行中';
          fill.style.transform = '';
          fill.classList.add('is-indeterminate');
        }
      });
    }

    async function createJob(url, format) {
      const res = await fetch('/api/jobs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url, format })
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || '这条链接暂时没有接住');
      return data;
    }

    async function poll(statusURL) {
      for (;;) {
        const res = await fetch(statusURL);
        const job = await res.json();
        if (!res.ok) throw new Error(job.error || '还没有听见回声');
        renderLinks(job);
        renderProgress(job);

        if (job.status === 'ready') {
          setLog('已经整理好啦，可以拿去分享了。', job);
          return;
        }
        if (job.status === 'failed') {
          setLog('这次没有整理成功。', job);
          throw new Error(job.error || '这次没有整理成功');
        }

        setLog(job.message || '正在轻轻整理中，请稍等...', job);
        await sleep(2500);
      }
    }

    form.addEventListener('submit', async event => {
      event.preventDefault();
      result.classList.remove('is-visible');
      resetJobProgress();
      setLink(direct, '', '正在寻找视频入口...');
      setLink(playback, '', '正在整理分享链接...');
      result.classList.add('is-visible');
      submit.disabled = true;
      submit.textContent = '整理中...';

      const url = document.querySelector('#url').value.trim();
      const format = new FormData(form).get('format');

      try {
        setLog('正在把这条链接放进小托盘...');
        const job = await createJob(url, format);
        setLog('已经收到啦，正在轻轻整理...', job);
        await poll(job.status_url.replace(location.origin, ''));
      } catch (error) {
        setLog(error.message);
      } finally {
        submit.disabled = false;
        submit.textContent = '轻轻整理';
      }
    });

    form.querySelectorAll('input[name="format"]').forEach(input => {
      input.addEventListener('change', syncFormatControl);
    });

    copyButtons.forEach(button => {
      button.addEventListener('click', async () => {
        const target = document.querySelector('#' + button.dataset.copyTarget);
        const href = target ? target.getAttribute('href') : '';
        if (!href) return;
        await navigator.clipboard.writeText(href);
        button.textContent = '已复制';
        setTimeout(() => button.textContent = '复制', 1200);
      });
    });

    [direct, playback].forEach(anchor => {
      anchor.addEventListener('click', event => {
        const fullURL = anchor.dataset.fullUrl || '';
        if (!fullURL) return;
        event.preventDefault();
        anchor.dataset.expanded = anchor.dataset.expanded === 'true' ? 'false' : 'true';
        setLink(anchor, fullURL, '');
      });
    });

    window.addEventListener('scroll', updateProgress, { passive: true });
    window.addEventListener('resize', updateProgress);
    syncFormatControl();
    updateProgress();
  </script>
</body>
</html>
`
