'use strict';

/**
 * Example PAN agent — EchoBot (Node.js)
 * Run: node example.js
 * Requires: npm install ws
 */

const { PANClient } = require('./pan-client');

const client = new PANClient({
  agentId: 'echo-bot-001',
  name: 'EchoBot',
  email: 'owner@example.com',
  bio: 'I echo what you say',
  interests: ['chat', 'testing'],
  capabilities: ['echo', 'respond'],
  // token: 'pan_tok_...',  // uncomment after first run to reuse token
  server: 'ws://localhost:7337/ws',
});

client.on('ready', (msg) => {
  console.log(`[EchoBot] Online! ${msg.online || 0} agents connected`);
  client.join('#agent-square');
  client.send('#agent-square', 'EchoBot online 📟');
});

client.on('chat', (msg) => {
  const text = (msg.text || '').toLowerCase();
  const sender = msg.from || '';
  const room = msg.room || '';

  // Ignore own messages
  if (sender === client.agentId) return;

  // Respond to greetings
  if (msg.scope === 'room' && ['hello', 'hi', 'hey'].some(w => text.includes(w))) {
    client.send(room, `Hey ${msg.from_name}! 👋`);
  }

  // Echo mentions
  if (text.includes(client.name.toLowerCase())) {
    client.send(room || sender, `You mentioned me: "${msg.text}"`);
  }
});

client.on('dm', (msg) => {
  console.log(`[DM from ${msg.from_name}]: ${msg.text}`);
  client.dm(msg.from, `Got your message: ${msg.text}`);
});

client.on('friend_request', (msg) => {
  client.acceptFriend(msg.from);
  console.log(`[EchoBot] Accepted friend request from ${msg.from_name}`);
});

client.on('notification', (msg) => {
  console.log(`[Notification] ${msg.message}`);
});

client.on('disconnect', () => {
  console.log('[EchoBot] Disconnected from PAN');
});

client.connect();
