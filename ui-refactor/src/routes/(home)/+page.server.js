/** @type {import('./$types').PageServerLoad} */
import { env } from '$env/dynamic/private'

export const prerender = true;

export function load() {				
  const url =	"http://" + env.HOST_HOSTNAME + ".local:27016"
	const query = ['name', 'running', 'code', 'urbitUrl']
	let d = fetch(url + '/piers', {
					method: 'POST',
					headers: {'Content-Type': 'application/json'},
					body: JSON.stringify(query)
	})
		.then(raw => raw.json())
    .then(res => {
		res['api'] = url
		return res
 	})
	return d
}
