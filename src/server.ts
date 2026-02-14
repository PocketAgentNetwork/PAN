// -----------------------------------------------------------
// 🦉 A2A Network Server (TypeScript)
// -----------------------------------------------------------

import { WebSocketServer } from 'ws';
import { v4 as uuidv4 } from 'uuid';
import 'colors';
import { Config } from './config/index.js';
import { handleMessage, handleDisconnect } from './handlers/messageHandler.js';
import { ExtendedWebSocket } from './types/index.js';

const wss = new WebSocketServer({ port: Config.PORT });

console.log(`🦉 A2A Network v2 (TS) listening on port ${Config.PORT}...`.green.bold);
console.log(`Waiting for agents to connect...`.dim);

wss.on('connection', (ws: ExtendedWebSocket) => {
    // 1. Initialize State
    ws.id = uuidv4();
    ws.agentName = "Anonymous";
    ws.isAuthed = false;
    ws.rateLimit = { count: 0, start: Date.now() };
    ws.joinedRooms = new Set();

    console.log(`[+] New Connection: ${ws.id}`.yellow);

    // 2. Handle Messages
    ws.on('message', (message: string) => {
        handleMessage(ws, message.toString());
    });

    // 3. Handle Disconnect
    ws.on('close', () => {
        handleDisconnect(ws);
        console.log(`[-] Connection closed: ${ws.id}`.dim);
    });

    // 4. Handle Errors
    ws.on('error', (err) => {
        console.error(`[!] Socket Error: ${err.message}`.red);
    });
});
