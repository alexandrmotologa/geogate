import http from 'node:http';
import { spawn, execSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import puppeteer from 'puppeteer-core';

const ROOT_DIR = 'B:\\workgit\\geogate';
const DOCS_IMAGES = path.join(ROOT_DIR, 'docs', 'images');
const TEMP_FRAMES = path.join(ROOT_DIR, 'temp_frames');
const CHROME_PATH = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const FFMPEG_PATH = 'C:\\Users\\alexander\\AppData\\Local\\Microsoft\\WinGet\\Links\\ffmpeg.exe';

if (!fs.existsSync(TEMP_FRAMES)) {
  fs.mkdirSync(TEMP_FRAMES, { recursive: true });
}

async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function main() {
  console.log('1. Starting mock upstream server on port 3000...');
  const mockServer = http.createServer((req, res) => {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({
      status: 'ok',
      upstream: 'Mock Production App',
      path: req.url,
      country: req.headers['cf-ipcountry'] || 'US',
      city: req.headers['x-geo-city'] || 'Unknown',
      asn: req.headers['x-asn'] || 'AS0',
      security: req.headers['x-geo-security'] || 'clear',
    }));
  });

  await new Promise(resolve => mockServer.listen(3000, '127.0.0.1', resolve));
  console.log('Mock upstream server listening on 127.0.0.1:3000');

  console.log('2. Starting geogate.exe on port 8089...');
  const geogateProc = spawn(
    path.join(ROOT_DIR, 'geogate.exe'),
    ['serve', '-port', '8089', '-upstream', 'http://127.0.0.1:3000'],
    { stdio: 'inherit' }
  );

  await sleep(2500);

  console.log('3. Sending simulated proxy requests to populate Live Proxy Stream...');
  const testRequests = [
    '/api/checkout?simulate_country=JP',
    '/api/pricing?simulate_country=RO&simulate_city=Bucharest',
    '/dashboard?simulate_country=DE&simulate_latency=45ms',
    '/api/auth/login?simulate_country=GB&simulate_conn=tor',
    '/api/v2/stream?simulate_country=US&simulate_conn=datacenter',
    '/api/orders?simulate_country=FR',
    '/api/profile?simulate_country=BR',
    '/api/cart?simulate_country=CA',
    '/api/products?simulate_country=AU',
    '/api/status?simulate_country=IN',
    '/api/feed?simulate_country=ES',
    '/api/telemetry?simulate_country=NL',
    '/api/billing?simulate_country=SG',
    '/api/webhook?simulate_country=SE',
  ];

  for (const reqPath of testRequests) {
    try {
      await fetch(`http://127.0.0.1:8089${reqPath}`);
      await sleep(80);
    } catch (e) {
      console.warn(`Request to ${reqPath} failed:`, e.message);
    }
  }

  console.log('4. Launching headless Chrome via puppeteer-core...');
  const browser = await puppeteer.launch({
    executablePath: CHROME_PATH,
    headless: 'new',
    args: [
      '--no-sandbox',
      '--disable-setuid-sandbox',
      '--disable-gpu',
      '--font-render-hinting=none',
      '--window-size=1440,900',
    ],
    defaultViewport: { width: 1440, height: 900, deviceScaleFactor: 2 },
  });

  const page = await browser.newPage();

  console.log('5. Navigating to http://127.0.0.1:8089/...');
  await page.goto('http://127.0.0.1:8089/', { waitUntil: 'networkidle0' });
  await sleep(2500);

  // Take geogate_studio.png (Desktop 1440x950)
  console.log('6. Capturing geogate_studio.png...');
  await page.setViewport({ width: 1440, height: 950, deviceScaleFactor: 2 });
  await sleep(1500);
  await page.screenshot({ path: path.join(DOCS_IMAGES, 'geogate_studio.png') });
  console.log('Saved geogate_studio.png');

  // Switch to Live Proxy Stream
  console.log('7. Switching to Live Proxy Stream tab...');
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const btn = buttons.find(b => b.textContent && b.textContent.includes('Live Proxy Stream'));
    if (btn) btn.click();
  });
  await sleep(1500);

  // Click on "Inspect Header Diff" button to reveal Header Diff drawer
  await page.evaluate(() => {
    const inspectBtns = Array.from(document.querySelectorAll('button[title="Inspect Header Diff"]'));
    if (inspectBtns.length > 0) {
      inspectBtns[0].click();
    } else {
      const rows = Array.from(document.querySelectorAll('tbody tr'));
      if (rows.length > 0) rows[0].dispatchEvent(new MouseEvent('click', { bubbles: true }));
    }
  });
  await sleep(1200);
  await page.screenshot({ path: path.join(DOCS_IMAGES, 'geogate_proxy_stream.png') });
  console.log('Saved geogate_proxy_stream.png');

  // Switch to Batch Pipeline
  console.log('8. Switching to Batch Pipeline tab...');
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const btn = buttons.find(b => b.textContent && b.textContent.includes('Batch Pipeline'));
    if (btn) btn.click();
  });
  await sleep(1000);

  // Select preset "250 Benchmark IPs"
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const preset = buttons.find(b => b.textContent && (b.textContent.includes('250 Benchmark IPs') || b.textContent.includes('250')));
    if (preset) preset.click();
  });
  await sleep(500);

  // Click Execute Parallel Batch Lookup
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const runBtn = buttons.find(b => b.textContent && (b.textContent.includes('Execute Parallel Batch Lookup') || b.textContent.includes('Execute')));
    if (runBtn) runBtn.click();
  });
  await sleep(2500);
  await page.screenshot({ path: path.join(DOCS_IMAGES, 'geogate_batch.png') });
  console.log('Saved geogate_batch.png');

  // Switch to Geofence Matrix
  console.log('9. Switching to Geofence Matrix tab...');
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const btn = buttons.find(b => b.textContent && b.textContent.includes('Geofence Matrix'));
    if (btn) btn.click();
  });
  await sleep(1000);

  // Click preset Strict EU
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const preset = buttons.find(b => b.textContent && (b.textContent.includes('Strict EU') || b.textContent.includes('GDPR')));
    if (preset) preset.click();
  });
  await sleep(500);

  // Click Evaluate Geofence Rules
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const evalBtn = buttons.find(b => b.textContent && b.textContent.includes('Evaluate'));
    if (evalBtn) evalBtn.click();
  });
  await sleep(1500);
  await page.screenshot({ path: path.join(DOCS_IMAGES, 'geogate_geofence.png') });
  console.log('Saved geogate_geofence.png');

  // Mobile screenshot (390x844)
  console.log('10. Capturing geogate_mobile.png (390x844)...');
  await page.setViewport({ width: 390, height: 844, isMobile: true, hasTouch: true, deviceScaleFactor: 2 });
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const mapBtn = buttons.find(b => b.textContent && b.textContent.includes('World Map'));
    if (mapBtn) mapBtn.click();
  });
  await sleep(2000);
  await page.screenshot({ path: path.join(DOCS_IMAGES, 'geogate_mobile.png') });
  console.log('Saved geogate_mobile.png');

  // Render high-res 1024x1024 logo.png from logo.svg
  console.log('11. Rendering logo.png (1024x1024) from logo.svg...');
  const logoSvgPath = 'file:///' + path.join(DOCS_IMAGES, 'logo.svg').replace(/\\/g, '/');
  const logoPage = await browser.newPage();
  await logoPage.setViewport({ width: 1024, height: 1024, deviceScaleFactor: 1 });
  await logoPage.goto(logoSvgPath, { waitUntil: 'networkidle0' });
  await sleep(500);
  await logoPage.screenshot({ path: path.join(DOCS_IMAGES, 'logo.png'), omitBackground: false });
  await logoPage.close();
  console.log('Saved logo.png');

  // Animated Demo GIF generation
  console.log('12. Capturing animated frames for geogate_demo.gif...');
  await page.setViewport({ width: 1280, height: 768, isMobile: false, deviceScaleFactor: 1 });

  // Clean temp frames directory
  for (const f of fs.readdirSync(TEMP_FRAMES)) {
    fs.unlinkSync(path.join(TEMP_FRAMES, f));
  }

  let frameIdx = 0;
  async function snap(count = 1, delay = 200) {
    for (let c = 0; c < count; c++) {
      const framePath = path.join(TEMP_FRAMES, `frame_${String(frameIdx++).padStart(4, '0')}.png`);
      await page.screenshot({ path: framePath });
      await sleep(delay);
    }
  }

  // Go to World Map
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const mapBtn = buttons.find(b => b.textContent && b.textContent.includes('World Map'));
    if (mapBtn) mapBtn.click();
  });
  await sleep(1000);
  await snap(6, 180);

  // Type Bucharest in search input or click Bucharest if available
  await page.evaluate(() => {
    const searchInput = document.querySelector('input[placeholder*="Search"]');
    if (searchInput) {
      searchInput.value = 'Bucharest';
      searchInput.dispatchEvent(new Event('input', { bubbles: true }));
    }
  });
  await sleep(600);
  await snap(4, 150);

  // Click Bucharest from dropdown or search result
  await page.evaluate(() => {
    const items = Array.from(document.querySelectorAll('div, button, span'));
    const bucharest = items.find(el => el.textContent && el.textContent.trim() === 'Bucharest');
    if (bucharest) bucharest.click();
  });
  await sleep(1000);
  await snap(8, 200);

  // Switch to Live Proxy Stream
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const btn = buttons.find(b => b.textContent && b.textContent.includes('Live Proxy Stream'));
    if (btn) btn.click();
  });
  await sleep(1000);
  await snap(6, 180);

  // Click first transaction to open Header Diff drawer
  await page.evaluate(() => {
    const inspectBtn = document.querySelector('button[title="Inspect Header Diff"]');
    if (inspectBtn) {
      inspectBtn.click();
    } else {
      const rows = Array.from(document.querySelectorAll('tbody tr'));
      if (rows.length > 0) rows[0].dispatchEvent(new MouseEvent('click', { bubbles: true }));
    }
  });
  await sleep(1000);
  await snap(8, 200);

  // Switch to Batch Pipeline
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const btn = buttons.find(b => b.textContent && b.textContent.includes('Batch Pipeline'));
    if (btn) btn.click();
  });
  await sleep(1000);
  await snap(4, 180);

  // Load preset and Run Batch
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const preset = buttons.find(b => b.textContent && (b.textContent.includes('250 Benchmark IPs') || b.textContent.includes('250')));
    if (preset) preset.click();
  });
  await sleep(400);
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const runBtn = buttons.find(b => b.textContent && (b.textContent.includes('Execute Parallel Batch Lookup') || b.textContent.includes('Execute')));
    if (runBtn) runBtn.click();
  });
  await sleep(1500);
  await snap(8, 200);

  // Switch to Geofence Matrix
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const btn = buttons.find(b => b.textContent && b.textContent.includes('Geofence Matrix'));
    if (btn) btn.click();
  });
  await sleep(1000);
  await snap(4, 180);

  // Evaluate geofence
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const evalBtn = buttons.find(b => b.textContent && b.textContent.includes('Evaluate'));
    if (evalBtn) evalBtn.click();
  });
  await sleep(1200);
  await snap(8, 200);

  console.log(`Captured ${frameIdx} frames. Compiling with ffmpeg...`);
  const palettePath = path.join(TEMP_FRAMES, 'palette.png');
  const gifPath = path.join(DOCS_IMAGES, 'geogate_demo.gif');

  // Generate palette
  execSync(
    `"${FFMPEG_PATH}" -y -framerate 4 -i "${TEMP_FRAMES}\\frame_%04d.png" -vf "fps=4,scale=1024:-1:flags=lanczos,palettegen" "${palettePath}"`,
    { stdio: 'inherit' }
  );

  // Generate gif with palette
  execSync(
    `"${FFMPEG_PATH}" -y -framerate 4 -i "${TEMP_FRAMES}\\frame_%04d.png" -i "${palettePath}" -lavfi "fps=4,scale=1024:-1:flags=lanczos [x]; [x][1:v] paletteuse=dither=bayer:bayer_scale=3" "${gifPath}"`,
    { stdio: 'inherit' }
  );

  console.log('GIF compiled successfully at:', gifPath);

  // Cleanup
  console.log('13. Cleaning up...');
  await browser.close();
  geogateProc.kill();
  mockServer.close();

  // Remove temp frames
  for (const f of fs.readdirSync(TEMP_FRAMES)) {
    fs.unlinkSync(path.join(TEMP_FRAMES, f));
  }
  fs.rmdirSync(TEMP_FRAMES);

  console.log('All done successfully!');
}

main().catch(err => {
  console.error('Fatal error in capture script:', err);
  process.exit(1);
});
