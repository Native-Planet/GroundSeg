import { writable } from 'svelte/store'

//
// fade transition params
//
export const fadeIn = {duration:200, delay: 160}
export const fadeOut = {duration:200}

//
// graphql queries
//
export const homepageQuery = () => {
	return "query { piers {name,running,code} }"
}

//
// writable stores
//
export const piers = writable({})
export const system = writable({})
export const api = writable('')

//
// state update main functions
//
export const updateState = update => {
	updatePiers(update['piers'])
	updateApi(update['api'])
}

//export const updateApi = a => {if(a){api.set(a)}}

export const updatePiers = p => {
	if (p) {piers.update( s => {
			s = p
			return s
	})}
}
/*
export const updateSystem = update => {
	if (update['system']) {
		state.update( s => {
			s['system'] = update['system']
			return s
	})}
}
*/

