import Urbit from '@urbit/http-api'

const toResolvedResult = (ok, error = null) => {
  return {
    ok,
    error
  }
}

export const createUrbitSseClient = (baseUrl = '') => {
  const urbit = new Urbit(baseUrl)

  const sendPoke = (mark, json) => {
    return new Promise(resolve => {
      urbit.poke({
        app: 'groundseg',
        mark,
        json,
        onSuccess: () => resolve(toResolvedResult(true)),
        onError: error => resolve(toResolvedResult(false, error))
      })
    })
  }

  const sendAction = payload => {
    return sendPoke('action', { action: JSON.stringify({ payload }) })
  }

  const sendHeartbeat = () => {
    return sendPoke('heartbeat', { action: '' })
  }

  const subscribe = ({ ship, onOpen, onRetry, onError, onEvent, onQuit, onSubscriptionError }) => {
    urbit.ship = ship

    urbit.onOpen = () => {
      if (onOpen) {
        onOpen()
      }
    }

    urbit.onRetry = () => {
      if (onRetry) {
        onRetry()
      }
    }

    urbit.onError = error => {
      if (onError) {
        onError(error)
      }
    }

    urbit.subscribe({
      app: 'groundseg',
      path: '/broadcast',
      event: event => {
        if (onEvent) {
          onEvent(event)
        }
      },
      quit: () => {
        if (onQuit) {
          onQuit()
        }
      },
      err: error => {
        if (onSubscriptionError) {
          onSubscriptionError(error)
        }
      }
    })
  }

  return {
    sendAction,
    sendHeartbeat,
    subscribe
  }
}
