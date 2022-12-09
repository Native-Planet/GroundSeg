/** @type {import('./$types').PageServerLoad} */
import { env } from '$env/dynamic/private'

export const prerender = false;

export function load({ cookies }) {				
  const sessionid = cookies.get('sessionid');

  let url = "http://" + env.HOST_HOSTNAME + ".local:27016"
  let query = 'http://127.0.0.1:27016/system?sessionid=' + sessionid 

	let d = fetch(query, {credentials:"include"})
	.then(j => j.json())
  .then(r => {
    if (r == 404) {return {api:url,status:r}}
    else {r['api'] = url; return r}
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
