package main

const uiHTML = `<!DOCTYPE html>
<html lang="de">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Galaxis DevCtl</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: 'JetBrains Mono', 'Fira Code', ui-monospace, monospace;
    font-size: 13px;
    background: #050a0f;
    color: #cbd5e1;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }
  .header {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 20px;
    background: #000;
    border-bottom: 1px solid #1e293b;
    flex-shrink: 0;
  }
  .header-title { font-size: 11px; font-weight: bold; letter-spacing: .12em; text-transform: uppercase; color: #10b981; }
  .header-sub   { font-size: 11px; color: #475569; }
  .header-time  { margin-left: auto; font-size: 11px; color: #334155; }

  .components {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
    gap: 12px;
    padding: 16px 20px;
    flex-shrink: 0;
  }
  .card {
    background: #0f172a;
    border: 1px solid #1e293b;
    border-radius: 6px;
    padding: 14px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .card.running  { border-color: #064e3b; }
  .card.starting { border-color: #78350f; }
  .card.error    { border-color: #7f1d1d; }

  .card-header { display: flex; align-items: center; gap: 8px; }
  .dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
  .dot.running  { background: #10b981; box-shadow: 0 0 6px #10b981; }
  .dot.starting { background: #f59e0b; animation: pulse 1s infinite; }
  .dot.stopped  { background: #374151; }
  .dot.error    { background: #ef4444; }
  @keyframes pulse { 0%,100%{opacity:1} 50%{opacity:.4} }

  .card-name { font-weight: bold; font-size: 12px; color: #e2e8f0; flex: 1; }
  .badge { font-size: 10px; padding: 1px 5px; border-radius: 3px; font-weight: bold;
           letter-spacing: .05em; text-transform: uppercase; }
  .badge.running  { background: #064e3b; color: #6ee7b7; }
  .badge.starting { background: #78350f; color: #fcd34d; }
  .badge.stopped  { background: #1e293b; color: #64748b; }
  .badge.error    { background: #7f1d1d; color: #fca5a5; }

  .card-meta { display: grid; grid-template-columns: auto 1fr; gap: 2px 12px;
               font-size: 11px; }
  .card-meta .lbl { color: #334155; }
  .card-meta .val { color: #94a3b8; }
  .card-meta .val.ok  { color: #6ee7b7; }
  .card-meta .val.err { color: #fca5a5; font-size: 10px; word-break: break-all; }

  .card-actions { display: flex; gap: 6px; margin-top: 2px; }
  button {
    cursor: pointer; border: 1px solid; border-radius: 4px;
    font-family: inherit; font-size: 11px; padding: 3px 10px;
    transition: all .15s; letter-spacing: .03em;
  }
  .btn-start   { border-color: #065f46; color: #34d399; background: transparent; }
  .btn-start:hover:not(:disabled)   { background: #064e3b; }
  .btn-stop    { border-color: #7f1d1d; color: #f87171; background: transparent; }
  .btn-stop:hover:not(:disabled)    { background: #450a0a; }
  .btn-restart { border-color: #78350f; color: #fcd34d; background: transparent; }
  .btn-restart:hover:not(:disabled) { background: #451a03; }
  .btn-logs    { border-color: #1e3a5f; color: #60a5fa; background: transparent; margin-left: auto; }
  .btn-logs:hover  { background: #0c1a2e; }
  .btn-logs.active { background: #0c1a2e; border-color: #3b82f6; color: #93c5fd; }
  button:disabled { opacity: .35; cursor: not-allowed; }

  .log-panel {
    flex: 1; display: flex; flex-direction: column; min-height: 250px;
    border-top: 1px solid #1e293b; background: #000;
  }
  .log-bar {
    display: flex; align-items: center; gap: 8px; padding: 6px 20px;
    background: #050a0f; border-bottom: 1px solid #1e293b; flex-shrink: 0;
  }
  .log-bar-title { font-size: 11px; color: #475569; text-transform: uppercase; letter-spacing: .08em; }
  .log-bar-name  { font-size: 11px; color: #60a5fa; font-weight: bold; }
  .btn-clear { margin-left: auto; border-color: #1e293b; color: #475569; background: transparent;
               font-size: 10px; padding: 2px 8px; }
  .btn-clear:hover { border-color: #334155; color: #64748b; }

  .log-terminal {
    flex: 1; overflow-y: auto; padding: 10px 20px;
    font-size: 12px; line-height: 1.6;
  }
  .log-line        { white-space: pre-wrap; word-break: break-all; color: #64748b; }
  .log-line.devctl { color: #334155; font-style: italic; }
  .log-line.err    { color: #ef4444; }
  .log-line.ok     { color: #10b981; }
  .log-line.warn   { color: #f59e0b; }
  .log-empty       { color: #1e293b; font-style: italic; padding: 20px 0; }
  #log-end         { height: 1px; }
</style>
</head>
<body>

<div class="header">
  <span class="header-title">&#x2B21; Galaxis DevCtl</span>
  <span class="header-sub">Prozessmanager</span>
  <span class="header-time" id="clock"></span>
</div>

<div class="components" id="components"></div>

<div class="log-panel">
  <div class="log-bar">
    <span class="log-bar-title">Logs</span>
    <span class="log-bar-name" id="log-label">&#x2014; keine Auswahl &#x2014;</span>
    <button class="btn-clear" onclick="clearLogs()">Leeren</button>
  </div>
  <div class="log-terminal" id="log-terminal">
    <div class="log-empty">Auf eine Komponente klicken um Logs anzuzeigen.</div>
  </div>
  <div id="log-end"></div>
</div>

<script>
var ORDER = ['postgres', 'galaxis-api', 'galaxis-frontend']
var state = {}
var activeLog = null
var logEs = null

function fetchStatus() {
  fetch('/api/status').then(function(r){ return r.json() }).then(function(data){
    data.components.forEach(function(c){ state[c.id] = c })
    renderComponents()
  }).catch(function(){})
}

function renderComponents() {
  var el = document.getElementById('components')
  var html = ''
  ORDER.forEach(function(id) {
    var c = state[id]
    if (!c) return
    var st = c.status
    var running  = st === 'running'
    var stopped  = st === 'stopped' || st === 'error'
    var starting = st === 'starting'

    var meta = ''
    meta += '<span class="lbl">Port</span><span class="val">:' + c.port + '</span>'
    meta += '<span class="lbl">PID</span><span class="val">' + (c.pid || '&mdash;') + '</span>'
    meta += '<span class="lbl">Uptime</span><span class="val' + (running ? ' ok' : '') + '">' + (c.uptime || '&mdash;') + '</span>'
    if (c.error) {
      meta += '<span class="lbl">Fehler</span><span class="val err">' + esc(c.error) + '</span>'
    }

    var logCls = activeLog === id ? ' active' : ''

    html += '<div class="card ' + st + '" id="card-' + id + '">'
    html += '<div class="card-header">'
    html += '<div class="dot ' + st + '"></div>'
    html += '<span class="card-name">' + esc(c.display) + '</span>'
    html += '<span class="badge ' + st + '">' + st + '</span>'
    html += '</div>'
    html += '<div class="card-meta">' + meta + '</div>'
    html += '<div class="card-actions">'
    html += '<button class="btn-start" ' + (!stopped ? 'disabled' : '') + ' onclick="api(\'start\',\'' + id + '\')">Start</button>'
    html += '<button class="btn-stop" ' + (!running && !starting ? 'disabled' : '') + ' onclick="api(\'stop\',\'' + id + '\')">Stop</button>'
    html += '<button class="btn-restart" ' + (stopped ? 'disabled' : '') + ' onclick="api(\'restart\',\'' + id + '\')">Restart</button>'
    html += '<button class="btn-logs' + logCls + '" onclick="openLogs(\'' + id + '\')">Logs</button>'
    html += '</div></div>'
  })
  el.innerHTML = html
}

function api(action, id) {
  fetch('/api/' + action + '/' + id, { method: 'POST' })
    .then(function(){ fetchStatus() }).catch(function(){})
}

function openLogs(id) {
  activeLog = id
  var c = state[id]
  document.getElementById('log-label').textContent = c ? c.display : id
  document.getElementById('log-terminal').innerHTML = ''
  if (logEs) { logEs.close(); logEs = null }

  logEs = new EventSource('/api/logs/' + id)
  logEs.onmessage = function(e) { appendLine(JSON.parse(e.data)) }
  logEs.onerror = function() {}
  renderComponents()
}

function appendLine(text) {
  var term = document.getElementById('log-terminal')
  var div = document.createElement('div')
  div.className = 'log-line' + lineClass(text)
  div.textContent = text
  term.appendChild(div)
  document.getElementById('log-end').scrollIntoView({ behavior: 'smooth' })
}

function lineClass(text) {
  if (text.indexOf('[devctl]') === 0) return ' devctl'
  var lo = text.toLowerCase()
  if (/error|fatal|panic|failed/.test(lo)) return ' err'
  if (/warn/.test(lo)) return ' warn'
  if (/ok|ready|bereit|connected|listening/.test(lo)) return ' ok'
  return ''
}

function clearLogs() {
  document.getElementById('log-terminal').innerHTML = ''
}

function esc(s) {
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')
}

function updateClock() {
  document.getElementById('clock').textContent =
    new Date().toLocaleTimeString('de-DE', { hour12: false })
}

fetchStatus()
setInterval(fetchStatus, 2000)
updateClock()
setInterval(updateClock, 1000)
</script>
</body>
</html>`
