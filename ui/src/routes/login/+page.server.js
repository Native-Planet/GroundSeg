/** @type {import('./$types').PageServerLoad} */
import { env } from '$env/dynamic/private'

export const prerender = false;

export function load({ cookies }) {				
  const sessionid = cookies.get('sessionid');

  let query = 'http://127.0.0.1:27016/cookies?sessionid=' + sessionid 

	let d = fetch(query)
    .then(j => j.json())
    .then(r => {return {status:r}})
    .catch(err => {return {status:err}})

	return d
}
