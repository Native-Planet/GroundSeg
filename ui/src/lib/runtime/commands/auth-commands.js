export const createAuthCommands = ({ sendTypeCommand, sendActionCommand }, { clearLoginError }) => {
  const verify = () => {
    return sendTypeCommand('verify')
  }

  const login = password => {
    clearLoginError()
    return sendTypeCommand('login', { password })
  }

  const logout = () => {
    return sendTypeCommand('logout')
  }

  const logoutAll = () => {
    return sendActionCommand('logout', 'everywhere')
  }

  const modifyPassword = (old, pwd) => {
    return sendActionCommand('password', 'modify', { old, password: pwd })
  }

  return {
    verify,
    login,
    logout,
    logoutAll,
    modifyPassword
  }
}
