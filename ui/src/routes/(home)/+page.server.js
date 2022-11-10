/** @type {import('./$types').PageServerLoad} */
import { env } from '$env/dynamic/private'
import { dev } from '$lib/api'

export const prerender = false;

export function load() {				
  let url =	"http://groundseg_api:27016"
  if (dev) {url = "http://" + env.HOST_HOSTNAME + ".local:27016" }

	let d = fetch(url + '/urbits')
		.then(raw => raw.json())
    .then(res => {
		res['api'] = "http://" + env.HOST_HOSTNAME + ".local:27016"
		return res
 	})
		.catch(err => console.log(err))
	return d
}
