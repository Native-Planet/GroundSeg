/** @type {import('./$types').PageServerLoad} */
import { env } from '$env/dynamic/private'
import { homepageQuery } from '$lib/api'

export const prerender = true;

export function load() {				
  const url =	"http://" + env.HOST_HOSTNAME + ".local:27016/graphql"
	const query = { "data": homepageQuery()}
	let d = fetch(url, {
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
