'use client'

import React, { useState } from 'react'
import { getVehicleInvoices, payInvoice } from '@/lib/api'
import type { Invoice, PaymentScenario } from '@/lib/types'

export default function MemberPage() {
  const [licensePlate, setLicensePlate] = useState('')
  const [invoices, setInvoices] = useState<Invoice[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [hasSearched, setHasSearched] = useState(false)

  // --- Modal Payment State ---
  const [selectedInvoice, setSelectedInvoice] = useState<Invoice | null>(null)
  const [scenario, setScenario] = useState<PaymentScenario>('success')
  const [paying, setPaying] = useState(false)
  const [paymentResult, setPaymentResult] = useState<{
    success: boolean
    message: string
    txId?: string
  } | null>(null)

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!licensePlate.trim()) return

    try {
      setLoading(true)
      setError(null)
      setHasSearched(true)
      const res = await getVehicleInvoices(licensePlate.toUpperCase().trim())
      setInvoices(res.data || [])
    } catch (err: any) {
      setError(err.message || 'Failed to fetch invoices for this vehicle.')
      setInvoices([])
    } finally {
      setLoading(false)
    }
  }

  const handleOpenPayModal = (inv: Invoice) => {
    setSelectedInvoice(inv)
    setScenario('success')
    setPaymentResult(null)
  }

  const handleCloseModal = () => {
    setSelectedInvoice(null)
    setPaymentResult(null)
  }

  const handleProcessPayment = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedInvoice) return

    try {
      setPaying(true)
      setPaymentResult(null)

      const res = await payInvoice(selectedInvoice.id, { scenario })
      
      // Update local invoice list status to match
      setInvoices((prev) =>
        prev.map((inv) =>
          inv.id === selectedInvoice.id
            ? {
                ...inv,
                status: 'PAID',
                transaction_id: res.transaction_id,
                paid_at: new Date().toISOString(),
              }
            : inv
        )
      )

      setPaymentResult({
        success: true,
        message: `Payment successful! Transaction ID: ${res.transaction_id}`,
        txId: res.transaction_id,
      })

      // Short delay, then close modal
      setTimeout(() => {
        handleCloseModal()
      }, 2000)

    } catch (err: any) {
      setPaymentResult({
        success: false,
        message: err.message || 'Payment was declined by the gateway.',
      })
    } finally {
      setPaying(false)
    }
  }

  // Calculate stats
  const unpaidCount = invoices.filter((inv) => inv.status === 'UNPAID').length
  const totalFines = invoices.reduce((sum, inv) => sum + inv.calculated_amount, 0)

  return (
    <div className="space-y-8 animate-fade-in">
      {/* Header Banner */}
      <div className="bg-gradient-to-r from-emerald-700 to-teal-900 text-white rounded-2xl p-6 md:p-8 shadow-md">
        <h1 className="text-2xl md:text-3xl font-extrabold tracking-tight">Member Portal</h1>
        <p className="mt-2 text-emerald-100 max-w-2xl text-sm md:text-base">
          Look up parking violations associated with your vehicle license plate, view calculation details, and settle payments immediately.
        </p>
      </div>

      {/* Lookup Form */}
      <div className="card max-w-xl mx-auto">
        <div className="card-header bg-gray-50">
          <h2 className="text-sm font-bold text-gray-800 uppercase tracking-wider">Vehicle Lookup</h2>
        </div>
        <form onSubmit={handleSearch} className="card-body flex gap-3">
          <div className="flex-1">
            <input
              type="text"
              required
              placeholder="Enter License Plate (e.g. B 1234 XYZ)"
              value={licensePlate}
              onChange={(e) => setLicensePlate(e.target.value)}
              className="input uppercase font-semibold text-center text-lg tracking-wider"
            />
          </div>
          <button type="submit" disabled={loading} className="btn-primary px-6">
            {loading ? 'Searching...' : 'Search'}
          </button>
        </form>
      </div>

      {/* Results View */}
      {hasSearched && (
        <div className="space-y-6">
          {error && <div className="alert-error max-w-xl mx-auto">{error}</div>}

          {/* Quick stats */}
          {!loading && invoices.length > 0 && (
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 max-w-4xl mx-auto">
              <div className="card p-4 bg-white flex flex-col items-center">
                <span className="text-xs text-gray-400 font-semibold uppercase">Total Violations</span>
                <span className="text-2xl font-black text-gray-950 mt-1">{invoices.length}</span>
              </div>
              <div className="card p-4 bg-white flex flex-col items-center">
                <span className="text-xs text-gray-400 font-semibold uppercase">Unpaid Fines</span>
                <span className="text-2xl font-black text-amber-600 mt-1">{unpaidCount}</span>
              </div>
              <div className="card p-4 bg-white flex flex-col items-center">
                <span className="text-xs text-gray-400 font-semibold uppercase">Total Fine Amount</span>
                <span className="text-2xl font-black text-rose-600 mt-1">
                  Rp {totalFines.toLocaleString('id-ID')}
                </span>
              </div>
            </div>
          )}

          {/* Invoices List */}
          {loading ? (
            <div className="flex justify-center py-12">
              <svg className="animate-spin h-8 w-8 text-emerald-600" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
            </div>
          ) : invoices.length === 0 ? (
            <div className="card max-w-xl mx-auto text-center py-12 text-gray-500">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth="1.5" stroke="currentColor" className="w-12 h-12 mx-auto text-gray-300 mb-3">
                <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <p className="font-bold text-gray-800 text-base">Clear Record!</p>
              <p className="text-sm text-gray-400 mt-1">No violations or invoices found for "{licensePlate.toUpperCase()}".</p>
            </div>
          ) : (
            <div className="max-w-4xl mx-auto space-y-4">
              <h3 className="text-base font-extrabold text-gray-800">Violation Records</h3>
              
              <div className="grid gap-4">
                {invoices.map((inv) => (
                  <div key={inv.id} className="card hover:shadow-md transition-shadow">
                    <div className="card-body flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
                      
                      {/* Left: Info */}
                      <div className="space-y-2">
                        <div className="flex items-center gap-2">
                          <span className="font-mono font-bold text-sm text-gray-900">{inv.invoice_no}</span>
                          <span
                            className={
                              inv.status === 'PAID'
                                ? 'badge-paid'
                                : inv.status === 'VOIDED'
                                ? 'badge-voided'
                                : 'badge-unpaid'
                            }
                          >
                            {inv.status}
                          </span>
                        </div>

                        <div className="grid grid-cols-2 gap-x-6 gap-y-1 text-xs text-gray-600">
                          <div>
                            <span className="font-semibold block text-gray-400">Type</span>
                            <span>{inv.violation?.violation_type?.replace(/_/g, ' ') || 'Parking Offense'}</span>
                          </div>
                          <div>
                            <span className="font-semibold block text-gray-400">Location</span>
                            <span>{inv.violation?.location || 'Unknown'}</span>
                          </div>
                          <div>
                            <span className="font-semibold block text-gray-400">Date</span>
                            <span>
                              {inv.violation?.timestamp
                                ? new Date(inv.violation.timestamp).toLocaleString()
                                : 'N/A'}
                            </span>
                          </div>
                          {inv.transaction_id && (
                            <div>
                              <span className="font-semibold block text-gray-400">Tx ID</span>
                              <span className="font-mono text-[10px] text-gray-500">{inv.transaction_id}</span>
                            </div>
                          )}
                        </div>
                      </div>

                      {/* Right: Actions */}
                      <div className="flex items-center gap-4 border-t sm:border-t-0 pt-3 sm:pt-0 justify-between sm:justify-end">
                        <div className="text-right">
                          <span className="text-xs text-gray-400 block font-semibold">Fine Amount</span>
                          <span className="text-base font-extrabold text-gray-900">
                            Rp {inv.calculated_amount.toLocaleString('id-ID')}
                          </span>
                        </div>
                        {inv.status === 'UNPAID' && (
                          <button
                            onClick={() => handleOpenPayModal(inv)}
                            className="btn-primary py-1.5 text-xs bg-emerald-600 hover:bg-emerald-700"
                          >
                            Pay Now
                          </button>
                        )}
                      </div>

                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* ================= PAYMENT MODAL ================= */}
      {selectedInvoice && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-gray-950/40 backdrop-blur-sm animate-fade-in">
          <div className="card w-full max-w-md bg-white shadow-xl animate-scale-up">
            <div className="card-header flex items-center justify-between border-b">
              <h3 className="font-bold text-gray-900">Settle Fine Statement</h3>
              <button
                onClick={handleCloseModal}
                className="text-gray-400 hover:text-gray-600 transition-colors"
              >
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth="2.5" stroke="currentColor" className="w-5 h-5">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <form onSubmit={handleProcessPayment} className="card-body space-y-5">
              {paymentResult && (
                <div
                  className={`text-xs p-3 rounded-lg border ${
                    paymentResult.success
                      ? 'bg-emerald-50 border-emerald-200 text-emerald-700'
                      : 'bg-red-50 border-red-200 text-red-700'
                  }`}
                >
                  {paymentResult.message}
                </div>
              )}

              {/* Billing Breakdown */}
              <div className="bg-gray-50 border rounded-lg p-3 space-y-2 text-xs">
                <span className="font-bold text-gray-800 uppercase block tracking-wider">
                  Calculation Formula Breakdown
                </span>
                
                <div className="flex justify-between">
                  <span className="text-gray-500">Base Fine Amount</span>
                  <span className="font-semibold text-gray-800">
                    Rp {selectedInvoice.calculated_amount.toLocaleString('id-ID')}
                  </span>
                </div>
                
                <div className="flex justify-between border-t pt-2">
                  <span className="text-gray-500 font-semibold">Total Amount Due</span>
                  <span className="font-extrabold text-emerald-700 text-sm">
                    Rp {selectedInvoice.calculated_amount.toLocaleString('id-ID')}
                  </span>
                </div>
              </div>

              {/* Required Scenario Selection */}
              <div>
                <label className="label text-xs">Test Gateway Scenario (Required)</label>
                <div className="grid grid-cols-2 gap-3 mt-1">
                  <label className={`flex items-center gap-2 p-2 border rounded-lg cursor-pointer transition-all text-xs font-semibold ${
                    scenario === 'success' ? 'border-emerald-500 bg-emerald-50 text-emerald-700' : 'border-gray-200 hover:bg-gray-50'
                  }`}>
                    <input
                      type="radio"
                      name="scenario"
                      value="success"
                      checked={scenario === 'success'}
                      onChange={() => setScenario('success')}
                      className="hidden"
                    />
                    <span>✅ Success</span>
                  </label>

                  <label className={`flex items-center gap-2 p-2 border rounded-lg cursor-pointer transition-all text-xs font-semibold ${
                    scenario === 'failed' ? 'border-rose-500 bg-rose-50 text-rose-700' : 'border-gray-200 hover:bg-gray-50'
                  }`}>
                    <input
                      type="radio"
                      name="scenario"
                      value="failed"
                      checked={scenario === 'failed'}
                      onChange={() => setScenario('failed')}
                      className="hidden"
                    />
                    <span>❌ Failed</span>
                  </label>
                </div>
              </div>

              <div className="flex gap-3 mt-4">
                <button
                  type="button"
                  onClick={handleCloseModal}
                  className="flex-1 btn-secondary text-xs"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={paying}
                  className="flex-1 btn-primary text-xs bg-emerald-600 hover:bg-emerald-700"
                >
                  {paying ? 'Processing...' : 'Settle Payment'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

    </div>
  )
}
