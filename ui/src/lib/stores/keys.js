import { get, writable } from 'svelte/store'
import { loadSession } from './gs-crypto'
import { wsPort } from './websocket'

const PENDING_KEY = 'groundseg:keys:pending'
const MIN_POLL_INTERVAL = 60
const MAX_POLL_INTERVAL = 300
const TERMINAL_STATUSES = new Set(['complete', 'confirmed', 'failed'])

export const keyPending = writable([])

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

export const keysRequest = async (path, payload = {}) => {
  const response = await fetch(`${apiBase()}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(await withToken(payload))
  })
  let body = {}
  try {
    body = await response.json()
  } catch (error) {
    throw new Error(`Keys request failed: ${response.status}`)
  }
  if (!response.ok || body.ok === false) {
    throw new Error(body.error || `Keys request failed: ${response.status}`)
  }
  return body
}

const readPending = () => {
  if (typeof localStorage === 'undefined') return []
  try {
    const parsed = JSON.parse(localStorage.getItem(PENDING_KEY) || '[]')
    return Array.isArray(parsed) ? parsed : []
  } catch (error) {
    return []
  }
}

const writePending = items => {
  keyPending.set(items)
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem(PENDING_KEY, JSON.stringify(items))
  }
}

export const loadKeyPending = () => {
  writePending(readPending())
}

export const addKeyPending = tx => {
  if (!tx) return
  const id = tx.hash || tx.signature || `${tx.ship}-${tx.operation}-${tx.submittedAt}`
  const next = [
    tx,
    ...get(keyPending).filter(item => (item.hash || item.signature || `${item.ship}-${item.operation}-${item.submittedAt}`) !== id)
  ]
  writePending(next.slice(0, 12))
}

export const removeKeyPending = tx => {
  const id = tx.hash || tx.signature || `${tx.ship}-${tx.operation}-${tx.submittedAt}`
  writePending(get(keyPending).filter(item => (item.hash || item.signature || `${item.ship}-${item.operation}-${item.submittedAt}`) !== id))
}

export const updateKeyPending = updated => {
  const id = updated.hash || updated.signature || `${updated.ship}-${updated.operation}-${updated.submittedAt}`
  writePending(get(keyPending).map(item => {
    const itemId = item.hash || item.signature || `${item.ship}-${item.operation}-${item.submittedAt}`
    return itemId === id ? updated : item
  }))
}

export const getPoint = (ship, roller = '') => keysRequest('/keys/point', { ship, roller })

export const checkPending = tx => keysRequest('/keys/point', {
  ship: tx.ship,
  hash: tx.hash,
  roller: tx.roller
})

export const generateKeyfile = payload => keysRequest('/keys/keyfile', payload)

export const generateCode = payload => keysRequest('/keys/code', payload)

export const submitKeyOperation = payload => keysRequest('/keys/operation', payload)

export const prepareWalletOperation = payload => keysRequest('/keys/wallet/prepare', payload)

export const submitWalletOperation = payload => keysRequest('/keys/wallet/submit', payload)

const containsPendingTx = (pending, tx) => {
  if (!Array.isArray(pending)) return false
  return pending.some(item => {
    const sig = item?.rawTx?.sig || item?.rawTx?.Sig
    const hash = item?.hash || item?.Hash
    return (tx.signature && sig === tx.signature) || (tx.hash && hash === tx.hash)
  })
}

const nextPollFromBatch = (batch, currentInterval = MIN_POLL_INTERVAL) => {
  const batchWait = Number(batch?.timeUntilNext)
  if (Number.isFinite(batchWait) && batchWait > 0) {
    return Math.max(MIN_POLL_INTERVAL, Math.min(MAX_POLL_INTERVAL, batchWait + 15))
  }
  return Math.max(MIN_POLL_INTERVAL, Math.min(MAX_POLL_INTERVAL, Math.ceil(currentInterval * 1.5)))
}

export const pollDueKeyPending = async (force = false) => {
  const now = Date.now()
  const items = get(keyPending)
  for (const tx of items) {
    if (!force && TERMINAL_STATUSES.has(tx.status)) continue
    if (!force && tx.nextPollAt && tx.nextPollAt > now) continue
    try {
      const result = await checkPending(tx)
      const status = result.status || ''
      const stillPending = containsPendingTx(result.pending, tx) || Boolean(result.pendingTx)
      if (status === 'confirmed' || status === 'failed' || (!stillPending && status !== 'pending' && status !== 'sending')) {
        const terminalStatus = status && status !== 'unknown' ? status : 'complete'
        updateKeyPending({ ...tx, status: terminalStatus, nextPollAt: 0 })
        continue
      }
      const pollInterval = nextPollFromBatch(result.batch, tx.pollInterval || MIN_POLL_INTERVAL)
      updateKeyPending({
        ...tx,
        status: status || 'pending',
        pollInterval,
        nextPollAt: Date.now() + pollInterval * 1000
      })
    } catch (error) {
      const pollInterval = Math.max(MIN_POLL_INTERVAL, Math.min(MAX_POLL_INTERVAL, (tx.pollInterval || MIN_POLL_INTERVAL) * 2))
      updateKeyPending({
        ...tx,
        status: 'checking',
        pollInterval,
        nextPollAt: Date.now() + pollInterval * 1000,
        lastError: error.message
      })
    }
  }
}

export const downloadText = (filename, text) => {
  const blob = new Blob([text], { type: 'text/plain;charset=utf-8' })
  const url = window.URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.style.display = 'none'
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  window.URL.revokeObjectURL(url)
}
