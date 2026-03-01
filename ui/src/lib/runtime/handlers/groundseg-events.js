import { connected, firstLoad, structure } from '../../stores/data.js'

export const parseBroadcastCord = cord => {
  try {
    return JSON.parse(cord)
  } catch (error) {
    console.error('Failed to parse:', error)
    return null
  }
}

export const routeGroundsegBroadcast = ({
  broadcast,
  onLoginActivity
}) => {
  if (!broadcast || typeof broadcast !== 'object') {
    return
  }

  if (broadcast.type === 'init') {
    connected.set(true)
    return
  }

  if (broadcast.type === 'structure') {
    structure.set(broadcast)
    firstLoad.set(false)
    return
  }

  if (broadcast.type === 'urbit-activity' && broadcast.payloadType === 'login') {
    if (onLoginActivity) {
      onLoginActivity(broadcast)
    }
  }
}
