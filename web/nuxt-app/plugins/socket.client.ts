import { io, Socket } from 'socket.io-client';
import { defineNuxtPlugin, useRuntimeConfig } from '#app';

export default defineNuxtPlugin(() => {
  const config = useRuntimeConfig();
  const socketUrl = config.public.socketUrl || '';
  const socket: Socket = io(socketUrl, {
    transports: ['websocket', 'polling']
  });

  return {
    provide: {
      socket
    }
  };
});
