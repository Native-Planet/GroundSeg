import { readable, writable } from 'svelte/store';
import { browser } from '$app/environment';

export const lastActivity = writable(new Date())

export const alive = readable({x:0, y:0}, (set) => {
  if (browser) {
    document.body.addEventListener("mousemove", move);
    
    function move(event) {
      set({
        x: event.clientX,
        y: event.clientY,
      });
    }
    return () => {
      document.body.removeEventListener("mousemove", move);
    }
	}
})
