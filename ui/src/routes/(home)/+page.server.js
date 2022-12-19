/** @type {import('./$types').PageServerLoad} */
import { env } from '$env/dynamic/private'

export const prerender = false;

export function load({ cookies }) {				
  const sessionid = cookies.get('sessionid');

  let query = 'http://127.0.0.1:27016/urbits?sessionid=' + sessionid 

	let d = fetch(query)
	.then(j => j.json())
  .then(r => {
    if ((r == 404) || (r == 'setup')) {return {status:r}}
    else {return r}
  })
  .catch(err => {
    console.log(err)
    if ((typeof err) == 'object') {
      err = 'noconn'
    }
    return {status:err}
  })

	return d
}
