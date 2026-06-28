import { get } from 'svelte/store'
import { URBIT_MODE } from './data'
import { loadSession } from './gs-crypto'
import { wsPort } from './websocket'

const apiBase = () => {
  const protocol = window.location.protocol === 'https:' ? 'https:' : 'http:'
  return `${protocol}//${window.location.hostname}:${get(wsPort)}`
}

const withToken = async payload => {
  const token = await loadSession()
  if (!token?.id || !token?.token) {
    throw new Error('GroundSeg login required')
  }
  return { ...payload, token }
}

export const configFileRequest = async payload => {
  if (get(URBIT_MODE)) {
    throw new Error('Config editor is only available from GroundSeg')
  }
  const response = await fetch(`${apiBase()}/config/files`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(await withToken(payload))
  })
  let body = {}
  try {
    body = await response.json()
  } catch (error) {
    throw new Error(`Config request failed: ${response.status}`)
  }
  if (!response.ok || body.ok === false) {
    throw new Error(body.error || `Config request failed: ${response.status}`)
  }
  return body
}

export const listConfigFiles = () => configFileRequest({ action: 'list' })

export const readConfigFile = file => configFileRequest({ action: 'read', file })

export const saveConfigFile = (file, content) => configFileRequest({ action: 'save', file, content })
