/**
 * @typedef {Object} RealtimeSendStatus
 * @property {boolean} ok
 * @property {'missing'|'not_open'|'send_failed'=} reason
 * @property {number|string=} readyState
 * @property {unknown=} error
 */

/**
 * @typedef {Object} RealtimeClient
 * @property {() => unknown} connect
 * @property {(code?: number) => void} disconnect
 * @property {(data: string) => RealtimeSendStatus} send
 * @property {() => boolean} isOpen
 * @property {() => number|string} getReadyState
 */

/**
 * This contract describes the runtime websocket adapter boundary used by stores.
 * Implementations must provide deterministic connect/send/readiness semantics.
 */
export const REALTIME_CLIENT_CONTRACT_VERSION = '1.1.0'
