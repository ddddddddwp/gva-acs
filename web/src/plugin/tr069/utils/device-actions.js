export const canIssueDeviceCommand = (row) => row?.online === true

export const deviceParameterSyncPayload = () => ({ paths: ['Device.'] })
