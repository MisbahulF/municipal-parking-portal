import axios, { AxiosError } from 'axios'
import type {
  CreateViolationRequest,
  CreateViolationResponse,
  GetInvoiceResponse,
  GetVehicleInvoicesResponse,
  HealthCheckResponse,
  PayInvoiceRequest,
  PayInvoiceResponse,
  PublishFineRuleRequest,
  PublishFineRuleResponse,
  FineRule,
  Invoice,
} from './types'

const apiClient = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080',
  headers: {
    'Content-Type': 'application/json',
    Accept: 'application/json',
  },
  timeout: 10000,
})

apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError<{ error?: string; detail?: string }>) => {
    const serverMessage =
      error.response?.data?.error ??
      error.response?.data?.detail ??
      error.message ??
      'An unexpected error occurred.'
    return Promise.reject(new Error(serverMessage))
  }
)

export default apiClient

export const checkHealth = async (): Promise<HealthCheckResponse> => {
  const { data } = await apiClient.get<HealthCheckResponse>('/health')
  return data
}

export const createViolation = async (
  payload: CreateViolationRequest
): Promise<CreateViolationResponse> => {
  const { data } = await apiClient.post<CreateViolationResponse>(
    '/api/v1/violations',
    payload
  )
  return data
}

export const getViolation = async (id: number) => {
  const { data } = await apiClient.get(`/api/v1/violations/${id}`)
  return data
}

export const getInvoice = async (id: number): Promise<GetInvoiceResponse> => {
  const { data } = await apiClient.get<GetInvoiceResponse>(
    `/api/v1/invoices/${id}`
  )
  return data
}

export const getAllInvoices = async (): Promise<{ data: Invoice[] }> => {
  const { data } = await apiClient.get<{ data: Invoice[] }>('/api/v1/invoices')
  return data
}

export const getVehicleInvoices = async (
  licensePlate: string
): Promise<GetVehicleInvoicesResponse> => {
  const { data } = await apiClient.get<GetVehicleInvoicesResponse>(
    `/api/v1/vehicles/${encodeURIComponent(licensePlate)}/invoices`
  )
  return data
}

export const publishFineRule = async (
  payload: PublishFineRuleRequest
): Promise<PublishFineRuleResponse> => {
  const { data } = await apiClient.post<PublishFineRuleResponse>(
    '/api/v1/fine-rules',
    payload
  )
  return data
}

export const getActiveFineRule = async (): Promise<{ data: FineRule }> => {
  const { data } = await apiClient.get<{ data: FineRule }>(
    '/api/v1/fine-rules/active'
  )
  return data
}

export const payInvoice = async (
  invoiceId: number,
  payload: PayInvoiceRequest
): Promise<PayInvoiceResponse> => {
  const { data } = await apiClient.post<PayInvoiceResponse>(
    `/api/v1/payments/${invoiceId}/pay`,
    payload
  )
  return data
}
