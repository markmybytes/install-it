import { status } from '@/wailsjs/go/models'

export type Command = {
  id: string | number
  name?: string
  groupName: string
  config: {
    program: string
    options: Array<string>
    minExeTime: number
    allowRtCodes: Array<number>
    incompatibles: Array<string | number>
    globalExclusive: boolean
  }
}

export type Process = {
  command: Command
  status: status.Status
  procId?: string
  result?: {
    lapse: number
    exitCode: number
    stdout: string
    stderr: string
    error: string
    aborted: boolean
  }
}
