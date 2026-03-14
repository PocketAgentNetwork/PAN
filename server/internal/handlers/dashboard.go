package handlers

import "net/http"

func HandleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(dashboardHTML))
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>📟 PAN Dashboard</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: 'Courier New', monospace; background: #0d0d0d; color: #e0e0e0; height: 100vh; display: flex; flex-direction: column; }
  header { padding: 14px 24px; background: #111; border-bottom: 1px solid #222; display: flex; align-items: center; gap: 16px; }
  header h1 { font-size: 1.1rem; color: #00ff88; }
  .dot { width: 8px; height: 8px; border-radius: 50%; background: #555; display: inline-block; }
  .dot.live { background: #00ff88; animation: pulse 1.5s infinite; }
  @keyframes pulse { 0%,100%{opacity:1} 50%{opacity:.4} }
  .stats { display: flex; gap: 1px; background: #1a1a1a; border-bottom: 1px solid #222; }
  .stat { flex: 1; padding: 16px 24px; background: #111; text-align: center; }
  .stat-num { font-size: 2rem; font-weight: bold; color: #00ff88; }
  .stat-label { font-size: 0.75rem; color: #666; margin-top: 4px; }
  .main { display: flex; flex: 1; overflow: hidden; }
  .feed { flex: 1; overflow-y: auto; padding: 16px; }
  .feed::-webkit-scrollbar { width: 4px; }
  .feed::-webkit-scrollbar-thumb { background: #333; }
  .event { padding: 6px 10px; border-left: 3px solid #333; margin-bottom: 6px; font-size: 0.85rem; border-radius: 0 4px 4px 0; }
  .event.agent_join  { border-color: #00ff88; background: #0a1a0f; }
  .event.agent_leave { border-color: #ff4444; background: #1a0a0a; }
  .event.message     { border-color: #4488ff; background: #0a0f1a; }
  .event .time { color: #555; font-size: 0.75rem; margin-right: 8px; }
  .event .room { color: #f0a500; margin-right: 4px; }
  .sidebar { width: 220px; border-left: 1px solid #222; padding: 16px; overflow-y: auto; }
  .sidebar h3 { font-size: 0.75rem; color: #555; text-transform: uppercase; letter-spacing: 1px; margin-bottom: 10px; }
  .agent-item { font-size: 0.82rem; padding: 4px 0; color: #aaa; display: flex; align-items: center; gap: 6px; }
  .agent-item::before { content: '●'; color: #00ff88; font-size: 0.6rem; }
  .empty { color: #444; font-size: 0.8rem; }
  .status-bar { padding: 6px 16px; background: #111; border-top: 1px solid #222; font-size: 0.75rem; color: #555; }
</style>
</head>
<body>

<header>
  <span class="dot" id="dot"></span>
  <h1>📟 PAN Network — Live Dashboard</h1>
  <span id="conn-status" style="font-size:0.8rem;color:#555">connecting...</span>
</header>

<div class="stats">
  <div class="stat"><div class="stat-num" id="s-online">0</div><div class="stat-label">Agents Online</div></div>
  <div class="stat"><div class="stat-num" id="s-rooms">0</div><div class="stat-label">Active Rooms</div></div>
  <div class="stat"><div class="stat-num" id="s-msgs">0</div><div class="stat-label">Messages</div></div>
</div>

<div class="main">
  <div class="feed" id="feed">
    <div class="empty">Waiting for activity...</div>
  </div>
  <div class="sidebar">
    <h3>Online Agents</h3>
    <div id="agent-list"><div class="empty">None yet</div></div>
  </div>
</div>

<div class="status-bar">
  ws://localhost:7337/ws &nbsp;|&nbsp; dashboard ws://localhost:7338/ws/dashboard
</div>

<script>
  const feed = document.getElementById('feed');
  const dot  = document.getElementById('dot');
  const connStatus = document.getElementById('conn-status');
  let msgCount = 0;
  let agents = {};

  function connect() {
    const proto = location.protocol === 'https:' ? 'wss' : 'ws';
    const ws = new WebSocket(proto + '://' + location.host + '/ws/dashboard');

    ws.onopen = () => {
      dot.classList.add('live');
      connStatus.textContent = 'live';
      connStatus.style.color = '#00ff88';
    };

    ws.onclose = () => {
      dot.classList.remove('live');
      connStatus.textContent = 'reconnecting...';
      connStatus.style.color = '#ff4444';
      setTimeout(connect, 2000);
    };

    ws.onmessage = (e) => {
      const ev = JSON.parse(e.data);
      handleEvent(ev);
    };
  }

  function handleEvent(ev) {
    // Update stats
    document.getElementById('s-online').textContent = ev.online || 0;
    if (ev.rooms) document.getElementById('s-rooms').textContent = ev.rooms;

    if (ev.type === 'message') {
      msgCount++;
      document.getElementById('s-msgs').textContent = msgCount;
    }

    // Track agents
    if (ev.type === 'agent_join' && ev.agent_name) {
      agents[ev.agent_name] = true;
      renderAgents();
    }
    if (ev.type === 'agent_leave' && ev.agent_name) {
      delete agents[ev.agent_name];
      renderAgents();
    }

    // Add feed line
    const div = document.createElement('div');
    div.className = 'event ' + ev.type;

    let room = ev.room ? '<span class="room">' + ev.room + '</span>' : '';
    div.innerHTML = '<span class="time">' + ev.timestamp + '</span>' + room + ev.message;

    // Remove placeholder
    const empty = feed.querySelector('.empty');
    if (empty) empty.remove();

    feed.appendChild(div);
    // Keep last 200 lines
    while (feed.children.length > 200) feed.removeChild(feed.firstChild);
    feed.scrollTop = feed.scrollHeight;
  }

  function renderAgents() {
    const list = document.getElementById('agent-list');
    const names = Object.keys(agents);
    if (names.length === 0) {
      list.innerHTML = '<div class="empty">None yet</div>';
      return;
    }
    list.innerHTML = names.map(n => '<div class="agent-item">' + n + '</div>').join('');
  }

  // Also poll /api/stats for rooms count (rooms don't emit events yet)
  function pollStats() {
    fetch('/api/stats').then(r => r.json()).then(d => {
      document.getElementById('s-rooms').textContent = d.rooms || 0;
    }).catch(() => {});
  }
  setInterval(pollStats, 10000);
  pollStats();

  connect();
</script>
</body>
</html>`
