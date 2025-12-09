import { io, Socket } from 'socket.io-client';
import { defineNuxtPlugin, useRuntimeConfig } from '#imports';

export default defineNuxtPlugin(() => {
  const config = useRuntimeConfig();
  const isDev = process.env.NODE_ENV === 'development';
  const browserOrigin = typeof window !== 'undefined' ? window.location.origin : '';
  const socketUrl = config.public.socketUrl || (isDev ? 'http://localhost:5001' : browserOrigin);
  const socket: Socket = io(socketUrl, {
    path: '/socket.io',
    transports: ['polling'], // 使用长轮询避免本地 Werkzeug WebSocket 500
    upgrade: false
  });

  return {
    provide: {
      socket
    }
  };
});
