export const createUrbitMaintenanceCommands = ({ sendUrbitCommand }) => {
  const setUrbitLoom = (patp, value) => {
    return sendUrbitCommand('loom', { patp, value })
  }

  const setUrbitSnapTime = (patp, value) => {
    return sendUrbitCommand('snaptime', { patp, value })
  }

  const setPackSchedule = (patp, frequency, intervalType, time, day, date) => {
    return sendUrbitCommand('schedule-pack', { patp, frequency, intervalType, time, day, date })
  }

  const pausePackSchedule = patp => {
    return sendUrbitCommand('pause-pack-schedule', { patp })
  }

  const toggleUrbitAlias = patp => {
    return sendUrbitCommand('toggle-alias', { patp })
  }

  const marsPack = patp => {
    return sendUrbitCommand('pack', { patp })
  }

  const urthPackMeld = patp => {
    return sendUrbitCommand('pack-meld', { patp })
  }

  const urbitChop = patp => {
    return sendUrbitCommand('chop', { patp })
  }

  const urbitRollChop = patp => {
    return sendUrbitCommand('roll-chop', { patp })
  }

  const toggleChopAfterVereUpdate = patp => {
    return sendUrbitCommand('toggle-chop-on-vere-update', { patp })
  }

  const installPenpaiCompanion = patp => {
    return sendUrbitCommand('install-penpai-companion', { patp })
  }

  const uninstallPenpaiCompanion = patp => {
    return sendUrbitCommand('uninstall-penpai-companion', { patp })
  }

  const installGallseg = patp => {
    return sendUrbitCommand('install-gallseg', { patp })
  }

  const deleteStartramService = (patp, service) => {
    return sendUrbitCommand('delete-service', { patp, service })
  }

  const uninstallGallseg = patp => {
    return sendUrbitCommand('uninstall-gallseg', { patp })
  }

  return {
    setUrbitLoom,
    setUrbitSnapTime,
    setPackSchedule,
    pausePackSchedule,
    toggleUrbitAlias,
    marsPack,
    urthPackMeld,
    urbitChop,
    urbitRollChop,
    toggleChopAfterVereUpdate,
    installPenpaiCompanion,
    uninstallPenpaiCompanion,
    installGallseg,
    deleteStartramService,
    uninstallGallseg
  }
}
