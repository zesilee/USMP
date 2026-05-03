import axios from 'axios'

export interface Log {
  id: string
  time: string
  device: string
  type: string
  status: 'success' | 'failed'
  user: string
  detail: string
}

export interface LogQueryParams {
  page?: number
  pageSize?: number
  keyword?: string
  startTime?: string
  endTime?: string
  status?: string
}

const api = axios.create({
  baseURL: '/api/logs',
  timeout: 15000
})

export async function getLogs(params: LogQueryParams = {}): Promise<{ data: Log[], total: number }> {
  const res = await api.get('', { params })
  return res.data
}
