const persistentArgRules = [
  { label: '-B/--bootstrap', aliases: ['-B', '--bootstrap'] },
  { label: '-c/--pier', aliases: ['-c', '--pier'] },
  { label: '-d/--daemon', aliases: ['-d', '--daemon'] },
  { label: '-G/--key-string', aliases: ['-G', '--key-string'] },
  { label: '-p/--port/--ames-port', aliases: ['-p', '--port', '--ames-port'] },
  { label: '--http-port', aliases: ['--http-port'] },
  { label: '--loom', aliases: ['--loom'] },
  { label: '--snap-time', aliases: ['--snap-time'] },
  { label: '--dirname', aliases: ['--dirname'] },
  { label: '--devmode', aliases: ['--devmode'] },
  { label: '-t', aliases: ['-t'] },
  { label: '-w/--name', aliases: ['-w', '--name'] },
  { label: '--bootstrap-url', aliases: ['--bootstrap-url'] },
  { label: '--prop-url', aliases: ['--prop-url'] },
  { label: '--prop-name', aliases: ['--prop-name'] },
]

const firstBootArgRules = [
  { label: '-G/--key-string', aliases: ['-G', '--key-string'] },
  { label: '-k/--key-file', aliases: ['-k', '--key-file'] },
  { label: '-p/--port/--ames-port', aliases: ['-p', '--port', '--ames-port'] },
  { label: '--http-port', aliases: ['--http-port'] },
  { label: '--loom', aliases: ['--loom'] },
  { label: '-t', aliases: ['-t'] },
  { label: '-w/--name', aliases: ['-w', '--name'] },
  { label: '-x', aliases: ['-x'] },
]

export const lintPersistentArgs = input => lintArgs(input, persistentArgRules)

export const lintFirstBootArgs = input => lintArgs(input, firstBootArgRules)

export const parseArgString = input => {
  input = input.trim()
  if (input.length < 1) {
    return { args: [], error: '' }
  }

  const args = []
  let current = ''
  let tokenStarted = false
  let inSingle = false
  let inDouble = false
  let escaped = false

  const flush = () => {
    if (!tokenStarted) {
      return
    }
    args.push(current)
    current = ''
    tokenStarted = false
  }

  for (const char of input) {
    if (escaped) {
      current += char
      escaped = false
      continue
    }
    if (inSingle) {
      if (char === "'") {
        inSingle = false
      } else {
        current += char
      }
      continue
    }
    if (inDouble) {
      if (char === '"') {
        inDouble = false
      } else if (char === '\\') {
        escaped = true
      } else {
        current += char
      }
      continue
    }

    if (/\s/.test(char)) {
      flush()
    } else if (char === "'") {
      inSingle = true
      tokenStarted = true
    } else if (char === '"') {
      inDouble = true
      tokenStarted = true
    } else if (char === '\\') {
      escaped = true
      tokenStarted = true
    } else {
      current += char
      tokenStarted = true
    }
  }

  if (escaped) {
    return { args: [], error: 'Invalid CLI flags: unterminated escape' }
  }
  if (inSingle || inDouble) {
    return { args: [], error: 'Invalid CLI flags: unterminated quote' }
  }

  flush()
  return { args, error: '' }
}

const lintArgs = (input, rules) => {
  const parsed = parseArgString(input)
  if (parsed.error.length > 0) {
    return {
      valid: false,
      message: parsed.error,
      blockedFlags: [],
      args: [],
    }
  }

  const blockedFlags = []
  for (const arg of parsed.args) {
    for (const rule of rules) {
      if (!rule.aliases.some(alias => tokenMatchesFlag(arg, alias))) {
        continue
      }
      if (!blockedFlags.includes(rule.label)) {
        blockedFlags.push(rule.label)
      }
      break
    }
  }

  if (blockedFlags.length > 0) {
    return {
      valid: false,
      message: `Not allowed here: ${blockedFlags.join(', ')}`,
      blockedFlags,
      args: parsed.args,
    }
  }

  return {
    valid: true,
    message: '',
    blockedFlags: [],
    args: parsed.args,
  }
}

const tokenMatchesFlag = (token, flag) => {
  if (token === flag || token.startsWith(flag + '=')) {
    return true
  }
  if (flag.startsWith('--')) {
    return false
  }
  return token.startsWith(flag) && token.length > flag.length
}
