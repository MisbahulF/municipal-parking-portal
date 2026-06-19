/**
 * TypeScript interfaces mirroring the Go shared/models package.
 * Keep these in sync with any structural changes on the backend.
 */

// ─── Enums ────────────────────────────────────────────────────────────────────

export type ViolationType =
  | 'EXPIRED_METER'
  | 'NO_PARKING_ZONE'
  | 'BLOCKING_HYDRANT'
  | 'DISABLED_SPOT'

export type InvoiceStatus = 'UNPAID' | 'PAID' | 'VOIDED'

export type LogicType = 'FLAT' | 'PROGRESSIVE'

export type PaymentScenario = 'success' | 'failed'

// ─── Domain Models ────────────────────────────────────────────────────────────

export interface Violation {
  id: number
  license_plate: string
  violation_type: ViolationType
  location: string
  timestamp: string        // ISO 8601 UTC
  photo_url?: string
  created_at: string
  updated_at: string
}

export interface FineRuleDetail {
  id: number
  fine_rule_id: number
  violation_type: ViolationType
  base_amount: number
  logic_type: LogicType
  time_window: number
  multiplier: number
}

export interface FineRule {
  id: number
  version: number
  is_active: boolean
  created_at: string
  details?: FineRuleDetail[]
}

export interface Invoice {
  id: number
  violation_id: number
  invoice_no: string
  applied_fine_rule_id: number
  calculated_amount: number
  status: InvoiceStatus
  transaction_id?: string  // populated on successful payment
  paid_at?: string
  created_at: string
  updated_at: string
  // Preloaded associations
  violation?: Violation
  applied_fine_rule?: FineRule
}

export interface PaymentAttempt {
  id: number
  invoice_id: number
  transaction_id: string
  charge_status: string    // "paid" | "failed"
  scenario: PaymentScenario
  amount: number
  attempted_at: string
}

// ─── API Request / Response shapes ───────────────────────────────────────────

/** POST /api/v1/violations */
export interface CreateViolationRequest {
  license_plate: string
  violation_type: ViolationType
  location: string
  timestamp: string        // ISO 8601
  photo_url?: string
}

export interface CreateViolationResponse {
  message: string
  data: Violation
}

/** POST /api/v1/fine-rules */
export interface PublishFineRuleRequest {
  details: {
    violation_type: ViolationType
    base_amount: number
    logic_type: LogicType
    time_window?: number
    multiplier?: number
  }[]
}

export interface PublishFineRuleResponse {
  message: string
  data: FineRule
}

/** GET /api/v1/invoices/:id */
export interface GetInvoiceResponse {
  data: Invoice
}

/** POST /api/v1/payments/:invoice_id/pay */
export interface PayInvoiceRequest {
  scenario: PaymentScenario
}

export interface PayInvoiceResponse {
  message: string
  charge_status: string
  transaction_id: string
  data?: Invoice           // present on success
  invoice_status?: string  // present on failure
}

/** GET /api/v1/vehicles/:plate/invoices */
export interface GetVehicleInvoicesResponse {
  license_plate: string
  count: number
  data: Invoice[]
}

/** GET /health */
export interface HealthCheckResponse {
  gateway: 'ok' | 'degraded'
  downstream: { service: string; status: string }[]
}
