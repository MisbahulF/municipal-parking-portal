/**
 * api.ts — Axios instance and typed API helpers.
 *
 * All frontend code should import from this module rather than calling
 * axios directly, so the base URL and error handling are centralised.
 *
 * Base URL: NEXT_PUBLIC_API_URL (default: http://localhost:8080)
 */

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

// ─── Axios Instance ───────────────────────────────────────────────────────────

const apiClient = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080',
  headers: {
    'Content-Type': 'application/json',
    Accept: 'application/json',
  },
  timeout: 10_000, // 10 s
})

// Response interceptor — normalise error messages across the app.
apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError<{ error?: string; detail?: string }>) => {
    const serverMessage =
      error.response?.data?.error ??
      error.response?.data?.detail ??
      error.message ??
      'An unexpected error occurred.'
    // Attach a friendly message so callers can display it directly.
    return Promise.reject(new Error(serverMessage))
  }
)

export default apiClient

// ─── Health ───────────────────────────────────────────────────────────────────

export const checkHealth = async (): Promise<HealthCheckResponse> => {
  const { data } = await apiClient.get<HealthCheckResponse>('/health')
  return data
}

// ─── Violations (Officer) ─────────────────────────────────────────────────────

/**
 * Flow 1 — Record a new parking violation.
 * Automatically triggers invoice generation on the backend.
 */
export const createViolation = async (
  payload: CreateViolationRequest
): Promise<CreateViolationResponse> => {
  const { data } = await apiClient.post<CreateViolationResponse>(
    '/api/v1/violations',
    payload
  )
  return data
}

/**
 * Fetch a single violation by ID.
 */
export const getViolation = async (id: number) => {
  const { data } = await apiClient.get(`/api/v1/violations/${id}`)
  return data
}

// ─── Invoices (Billing) ───────────────────────────────────────────────────────

/**
 * Fetch a full invoice with preloaded violation and fine rule.
 */
export const getInvoice = async (id: number): Promise<GetInvoiceResponse> => {
  const { data } = await apiClient.get<GetInvoiceResponse>(
    `/api/v1/invoices/${id}`
  )
  return data
}

/**
 * Fetch all invoices in the system.
 */
export const getAllInvoices = async (): Promise<{ data: Invoice[] }> => {
  const { data } = await apiClient.get<{ data: Invoice[] }>('/api/v1/invoices')
  return data
}

/**
 * Fetch all invoices for a given license plate (newest first).
 */
export const getVehicleInvoices = async (
  licensePlate: string
): Promise<GetVehicleInvoicesResponse> => {
  const { data } = await apiClient.get<GetVehicleInvoicesResponse>(
    `/api/v1/vehicles/${encodeURIComponent(licensePlate)}/invoices`
  )
  return data
}

/**
 * Flow 3 — Publish a new versioned fine ruleset.
 * Atomically deactivates the current active rule.
 */
export const publishFineRule = async (
  payload: PublishFineRuleRequest
): Promise<PublishFineRuleResponse> => {
  const { data } = await apiClient.post<PublishFineRuleResponse>(
    '/api/v1/fine-rules',
    payload
  )
  return data
}

/**
 * Fetch the currently active fine rule ruleset.
 */
export const getActiveFineRule = async (): Promise<{ data: FineRule }> => {
  const { data } = await apiClient.get<{ data: FineRule }>(
    '/api/v1/fine-rules/active'
  )
  return data
}

// ─── Payments (Member) ────────────────────────────────────────────────────────

/**
 * Flow 4 — Member pays an invoice.
 * Passes the scenario to the mock payment engine.
 *   scenario: "success" → invoice marked PAID, transaction_id stored.
 *   scenario: "failed"  → invoice stays UNPAID, 402 returned.
 */
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
