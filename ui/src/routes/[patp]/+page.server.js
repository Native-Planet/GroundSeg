/** @type {import('./$types').PageServerLoad} */
import { env } from '$env/dynamic/private'

export const prerender = false;

export function load({ cookies }) {				
  const sessionid = cookies.get('sessionid');

  let url = "http://" + env.HOST_HOSTNAME + ".local:27016"
  let query = 'http://localhost/cookies?sessionid=' + sessionid 

	let d = fetch(query, {credentials:"include"})
	.then(j => j.json())
  .then(r => {
    if (r == 404) {return {api:url,status:r}}
    else {return {api:url}}
  })
  .catch(err => {console.log(err); return {api:url}})

	return d
}
