// -----------------------------------------------------------
// 🦉 Agent Client Example (TypeScript)
// -----------------------------------------------------------

import WebSocket from 'ws';
import { v4 as uuidv4 } from 'uuid';
import { Config } from './config/index.js';

// Change this to your server IP (or use localhost for testing)
const SERVER_URL = `ws://localhost:${Config.PORT}`;

// Your Agent Identity
const MY_ID = 'agent-' + uuidv4().slice(0, 8);
const MY_NAME = 'Agent-Alpha-TS';

console.log(`🦉 Connecting to ${SERVER_URL} as ${MY_NAME} (${MY_ID})...`);

const ws = new WebSocket(SERVER_URL);

ws.on('open', () => {
    console.log('✅ Connected!');

    // 1. Authenticate immediately
    ws.send(JSON.stringify({
        type: 'auth',
        agentId: MY_ID,
        name: MY_NAME,
        token: Config.SECRET_KEY // Use shared secret
    }));

    // 2. Join #init
    ws.send(JSON.stringify({ type: 'join', room: '#init' }));

    // 3. Wait & Chat
    setTimeout(() => {
        console.log('🗣️ Sending hello...');
        ws.send(JSON.stringify({
            type: 'chat',
            to: '#init', // Send to room
            text: 'Hello from TypeScript World! 🦉'
        }));
    }, 2000);
});

ws.on('message', (data: any) => {
    const msg = JSON.parse(data.toString());
    console.log('📩 Received:', msg);
});

ws.on('close', () => {
    console.log('❌ Disconnected from network.');
});

ws.on('error', (err: Error) => {
    console.error('⚠️ Connection error:', err.message);
});
