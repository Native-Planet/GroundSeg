/** @type {import('./$types').PageServerLoad} */
import { env } from '$env/dynamic/private'

export const prerender = true;

export function load() {				
  const url =	"http://" + env.HOST_HOSTNAME + ".local:27016"
	let d = fetch(url + '/urbits')
		.then(raw => raw.json())
    .then(res => {
		res['api'] = url
		return res
 	})
		.catch(err => console.log(err))
	return d
}
