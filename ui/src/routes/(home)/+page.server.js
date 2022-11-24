/** @type {import('./$types').PageServerLoad} */
import { env } from '$env/dynamic/private'

export const prerender = false;

export function load({ cookies }) {				
  const sessionid = cookies.get('sessionid');

  let url = "http://" + env.HOST_HOSTNAME + ".local:27016"
  let query = 'http://localhost/urbits?sessionid=' + sessionid 

	let d = fetch(query, {credentials:"include"})
	.then(j => j.json())
  .then(r => {
    if (r == 404) {return {api:url,status:r}}
    else {r['api'] = url; return r}
  })
  .catch(err => {console.log(err); return {api:url}})

	return d
}
