/** @type {import('./$types').PageServerLoad} */
import { env } from '$env/dynamic/private'
import { dev } from '$lib/api'

export const prerender = false;

export function load({ cookies }) {				
  const sessionid = cookies.get('sessionid');

  let url =	"http://groundseg_api:27016"
  if (dev) {url = "http://" + env.HOST_HOSTNAME + ".local:27016" }

	let d = fetch(url + '/cookies?sessionid=' + sessionid, {
    credentials:"include"
  })
  .then(r => r.json())
  .then(d => { return d })

  return {"status":d,"api":"http://" + env.HOST_HOSTNAME + ".local:27016"}
}
