/** @type {import('./$types').PageServerLoad} */
import { env } from '$env/dynamic/private'

export const prerender = false;

export function load() {				
  const url =	"http://groundseg_api:27016"
	let d = fetch(url + '/anchor')
		.then(raw => raw.json())
    .then(res => {
		res['api'] = "http://" + env.HOST_HOSTNAME + ".local:27016"
		return res
 	})
		.catch(err => console.log(err))
	return d
}
