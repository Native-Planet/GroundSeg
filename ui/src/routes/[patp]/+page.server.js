/** @type {import('./$types').PageServerLoad} */
import { env } from '$env/dynamic/private'
import { page } from '$app/stores'

export const prerender = false;

export function load() {				
  const url =	"http://" + env.HOST_HOSTNAME + ".local:27016"
	return {"api":url}
}
