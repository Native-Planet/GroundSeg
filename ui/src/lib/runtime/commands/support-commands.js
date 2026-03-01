export const createSupportCommands = ({ sendActionCommand }) => {
  const submitReport = (contact, description, ships, cpuProfile, penpai) => {
    return sendActionCommand('support', 'bug-report', {
      contact,
      description,
      ships,
      cpu_profile: cpuProfile,
      penpai
    })
  }

  return {
    submitReport
  }
}
