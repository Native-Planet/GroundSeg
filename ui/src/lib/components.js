
  import Logo from '$lib/Logo.svelte'
  import SysInfo from '$lib/SysInfo.svelte'
  import Power from '$lib/Power.svelte'
  import Network from '$lib/Network.svelte'
  import MinIO from '$lib/MinIO.svelte'
  import Anchor from '$lib/Anchor.svelte'
  import ExportLogs from '$lib/ExportLogs.svelte'
  import Boot from '$lib/Boot.svelte'
  import Sigil from '$lib/Sigil.svelte'
  import NewPier from '$lib/NewPier.svelte'
  import DeleteWarning from '$lib/DeleteWarning.svelte'
  import PierCredentials from '$lib/PierCredentials.svelte'
  import Dropzone from '$lib/Dropzone.svelte'
  import Settings from '$lib/Settings.svelte'

  // Components used in each page in routes directory

  export const layout = {
    settings:Settings
  }

  export const settings = {
    logo:Logo,
    sysInfo:SysInfo,
    power:Power,
    network:Network,
    minIO:MinIO,
    anchor:Anchor,
    exportLogs:ExportLogs
  }

  export const home = {
    logo:Logo,
    sigil:Sigil,
    boot:Boot
  }

  export const newID = {
    logo:Logo,
    newPier:NewPier
  }

  export const existingID = {
    logo:Logo,
    dropzone:Dropzone
  }

  export const profile = {
    logo:Logo,
    warning:DeleteWarning,
    sigil:Sigil,
    credentials:PierCredentials
  }

  export const logs = {
    logo:Logo
  }
