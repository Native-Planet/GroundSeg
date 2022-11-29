/** @type {import('./$types').PageServerLoad} */
import { env } from '$env/dynamic/private'

export const prerender = false;

export function load({ cookies }) {				
  const sessionid = cookies.get('sessionid');

  let url = "http://" + env.HOST_HOSTNAME + ".local:27016"
  let query = 'http://127.0.0.1:27016/cookies?sessionid=' + sessionid 

	let d = fetch(query, {credentials:"include"})
	.then(j => j.json())
  .then(r => {return {api:url,status:r}})
  .catch(err => {console.log(err); return {api:url}})

	return d
}
