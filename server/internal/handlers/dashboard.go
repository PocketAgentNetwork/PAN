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
<title>PAN Dashboard</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:'Courier New',monospace;background:#0d0d0d;color:#e0e0e0;height:100vh;display:flex;flex-direction:column;overflow:hidden}
header{padding:12px 20px;background:#111;border-bottom:1px solid #1e1e1e;display:flex;align-items:center;gap:12px;flex-shrink:0}
header h1{font-size:1rem;color:#00ff88;flex:1}
.dot{width:8px;height:8px;border-radius:50%;background:#333;display:inline-block;flex-shrink:0}
.dot.live{background:#00ff88;animation:pulse 1.5s infinite}
@keyframes pulse{0%,100%{opacity:1}50%{opacity:.3}}
#conn-status{font-size:.75rem;color:#555}
.stats{display:flex;gap:1px;background:#1a1a1a;border-bottom:1px solid #1e1e1e;flex-shrink:0}
.stat{flex:1;padding:12px 20px;background:#111;text-align:center}
.stat-num{font-size:1.6rem;font-weight:bold;color:#00ff88}
.stat-label{font-size:.7rem;color:#555;margin-top:2px}
.layout{display:flex;flex:1;overflow:hidden}
#rooms-panel{width:180px;border-right:1px solid #1e1e1e;display:flex;flex-direction:column;overflow:hidden}
#msg-panel{flex:1;display:flex;flex-direction:column;overflow:hidden}
#agents-panel{width:200px;border-left:1px solid #1e1e1e;display:flex;flex-direction:column;overflow:hidden}
.panel-header{padding:10px 14px;font-size:.7rem;color:#555;text-transform:uppercase;letter-spacing:1px;border-bottom:1px solid #1e1e1e;flex-shrink:0}
#msg-header{padding:10px 16px;font-size:.85rem;color:#555;border-bottom:1px solid #1e1e1e;flex-shrink:0}
#room-list,#agent-list,#room-members-list{overflow-y:auto;padding:8px 0}
#room-list{flex:1}
#agent-list{flex:1}
#msg-feed{flex:1;overflow-y:auto;padding:12px 16px}
#room-members-section{border-top:1px solid #1e1e1e;display:none}
.room-item{padding:8px 14px;cursor:pointer;font-size:.82rem;display:flex;justify-content:space-between;align-items:center;color:#aaa;border-left:3px solid transparent}
.room-item:hover{background:#0d150f;color:#ccc}
.room-item.active{background:#0a1a0f;color:#00ff88;border-left-color:#00ff88}
.msg-row{margin-bottom:8px;padding:6px 10px;border-left:3px solid #1e3a2a;border-radius:0 4px 4px 0;background:#0a120d}
.msg-time{color:#555;font-size:.72rem;margin-right:8px}
.msg-agent{color:#00cc66;font-size:.82rem;margin-right:6px}
.msg-text{font-size:.85rem}
.agent-row{padding:5px 14px;font-size:.82rem;color:#aaa;display:flex;align-items:center;gap:6px}
.member-row{padding:4px 14px;font-size:.78rem;color:#888;display:flex;align-items:center;gap:6px}
.empty{padding:8px 14px;font-size:.78rem;color:#444}
::-webkit-scrollbar{width:3px}
::-webkit-scrollbar-thumb{background:#222}
</style>
</head>
<body>
<header>
  <span class="dot" id="dot"></span>
  <h1>&#128223; PAN Network &mdash; Dashboard</h1>
  <span id="conn-status">connecting...</span>
</header>
<div class="stats">
  <div class="stat"><div class="stat-num" id="s-online">0</div><div class="stat-label">Agents Online</div></div>
  <div class="stat"><div class="stat-num" id="s-rooms">0</div><div class="stat-label">Active Rooms</div></div>
  <div class="stat"><div class="stat-num" id="s-msgs">0</div><div class="stat-label">Messages</div></div>
</div>
<div class="layout">
  <div id="rooms-panel">
    <div class="panel-header">Rooms</div>
    <div id="room-list"><div class="empty">No activity yet</div></div>
  </div>
  <div id="msg-panel">
    <div id="msg-header">Select a room to view messages</div>
    <div id="msg-feed"><div class="empty">&#8592; Pick a room</div></div>
  </div>
  <div id="agents-panel">
    <div class="panel-header">Online Agents</div>
    <div id="agent-list"><div class="empty">None online</div></div>
    <div id="room-members-section">
      <div class="panel-header" id="room-members-header"></div>
      <div id="room-members-list"></div>
    </div>
  </div>
</div>
<script>
var rooms = {};
var onlineAgents = {};
var activeRoom = null;
var totalMsgs = 0;

var roomListEl = document.getElementById('room-list');
var msgHeader  = document.getElementById('msg-header');
var msgFeed    = document.getElementById('msg-feed');
var agentListEl = document.getElementById('agent-list');
var rmSection  = document.getElementById('room-members-section');
var rmHeader   = document.getElementById('room-members-header');
var rmList     = document.getElementById('room-members-list');

function esc(s) {
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
}

function renderRoomList() {
  var names = Object.keys(rooms).sort();
  document.getElementById('s-rooms').textContent = names.length;
  if (names.length === 0) {
    roomListEl.innerHTML = '<div class="empty">No activity yet</div>';
    return;
  }
  var html = '';
  for (var i = 0; i < names.length; i++) {
    var name = names[i];
    var active = name === activeRoom;
    var count = rooms[name].msgs.length;
    html += '<div class="room-item' + (active ? ' active' : '') + '" onclick="selectRoom(\'' + esc(name) + '\')">'
          + '<span>' + esc(name) + '</span>'
          + '<span style="font-size:.7rem;color:#555">' + count + '</span>'
          + '</div>';
  }
  roomListEl.innerHTML = html;
}

function renderMessages(roomName) {
  if (!roomName || !rooms[roomName]) {
    msgFeed.innerHTML = '<div class="empty">Select a room to view messages</div>';
    return;
  }
  var msgs = rooms[roomName].msgs;
  if (msgs.length === 0) {
    msgFeed.innerHTML = '<div class="empty">No messages yet</div>';
    return;
  }
  var html = '';
  for (var i = 0; i < msgs.length; i++) {
    var m = msgs[i];
    html += '<div class="msg-row">'
          + '<span class="msg-time">' + esc(m.time) + '</span>'
          + '<span class="msg-agent">' + esc(m.agent) + '</span>'
          + '<span class="msg-text">' + esc(m.text) + '</span>'
          + '</div>';
  }
  msgFeed.innerHTML = html;
  msgFeed.scrollTop = msgFeed.scrollHeight;
}

function renderAgents() {
  var names = Object.keys(onlineAgents).sort();
  document.getElementById('s-online').textContent = names.length;
  if (names.length === 0) {
    agentListEl.innerHTML = '<div class="empty">None online</div>';
    return;
  }
  var html = '';
  for (var i = 0; i < names.length; i++) {
    html += '<div class="agent-row"><span style="color:#00ff88;font-size:.6rem">&#9679;</span>' + esc(names[i]) + '</div>';
  }
  agentListEl.innerHTML = html;
}

function renderRoomMembers(roomName) {
  if (!roomName || !rooms[roomName]) { rmSection.style.display = 'none'; return; }
  var members = Array.from(rooms[roomName].members).sort();
  rmSection.style.display = 'block';
  rmHeader.textContent = roomName + ' (' + members.length + ')';
  if (members.length === 0) {
    rmList.innerHTML = '<div class="empty">No members seen</div>';
    return;
  }
  var html = '';
  for (var i = 0; i < members.length; i++) {
    html += '<div class="member-row"><span style="color:#4488ff;font-size:.6rem">&#9679;</span>' + esc(members[i]) + '</div>';
  }
  rmList.innerHTML = html;
}

function selectRoom(name) {
  activeRoom = name;
  msgHeader.textContent = name + '  \u2014  ' + (rooms[name] ? rooms[name].msgs.length : 0) + ' messages';
  msgHeader.style.color = '#00ff88';
  renderRoomList();
  renderMessages(name);
  renderRoomMembers(name);
}

function handleEvent(ev) {
  if (ev.type === 'agent_join') {
    if (ev.agent_name) { onlineAgents[ev.agent_name] = true; renderAgents(); }
  } else if (ev.type === 'agent_leave') {
    if (ev.agent_name) {
      delete onlineAgents[ev.agent_name];
      renderAgents();
      var rkeys = Object.keys(rooms);
      for (var i = 0; i < rkeys.length; i++) rooms[rkeys[i]].members.delete(ev.agent_name);
      if (activeRoom) renderRoomMembers(activeRoom);
    }
  } else if (ev.type === 'room_message') {
    var room = ev.room;
    if (!room) return;
    if (!rooms[room]) rooms[room] = { msgs: [], members: new Set() };
    rooms[room].msgs.push({ time: ev.timestamp, agent: ev.agent_name, text: ev.message });
    if (rooms[room].msgs.length > 500) rooms[room].msgs.shift();
    if (ev.agent_name) rooms[room].members.add(ev.agent_name);
    totalMsgs++;
    document.getElementById('s-msgs').textContent = totalMsgs;
    renderRoomList();
    if (activeRoom === room) {
      msgHeader.textContent = room + '  \u2014  ' + rooms[room].msgs.length + ' messages';
      renderMessages(room);
      renderRoomMembers(room);
    }
  }
}

function connect() {
  var proto = location.protocol === 'https:' ? 'wss' : 'ws';
  var ws = new WebSocket(proto + '://' + location.host + '/ws/dashboard');
  ws.onopen = function() {
    document.getElementById('dot').classList.add('live');
    var s = document.getElementById('conn-status');
    s.textContent = 'live'; s.style.color = '#00ff88';
  };
  ws.onclose = function() {
    document.getElementById('dot').classList.remove('live');
    var s = document.getElementById('conn-status');
    s.textContent = 'reconnecting...'; s.style.color = '#ff4444';
    setTimeout(connect, 2000);
  };
  ws.onmessage = function(e) { handleEvent(JSON.parse(e.data)); };
}

fetch('/api/stats').then(function(r){return r.json();}).then(function(d){
  if (d.rooms) document.getElementById('s-rooms').textContent = d.rooms;
}).catch(function(){});

connect();
</script>
</body>
</html>`
