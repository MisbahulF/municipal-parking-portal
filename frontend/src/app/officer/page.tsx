'use client'

import React, { useState, useEffect } from 'react'
import {
  createViolation,
  getActiveFineRule,
  publishFineRule,
  getVehicleInvoices,
} from '@/lib/api'
import type {
  ViolationType,
  LogicType,
  FineRule,
  Violation,
  Invoice,
} from '@/lib/types'

const VIOLATION_TYPES: { value: ViolationType; label: string }[] = [
  { value: 'EXPIRED_METER', label: 'Expired Meter' },
  { value: 'NO_PARKING_ZONE', label: 'No Parking Zone' },
  { value: 'BLOCKING_HYDRANT', label: 'Blocking Hydrant' },
  { value: 'DISABLED_SPOT', label: 'Disabled Spot' },
]

export default function OfficerPage() {
  // --- Active Fine Rules State ---
  const [activeRule, setActiveRule] = useState<FineRule | null>(null)
  const [rulesLoading, setRulesLoading] = useState(true)
  const [rulesError, setRulesError] = useState<string | null>(null)

  // --- Rule Editor State ---
  const [editorDetails, setEditorDetails] = useState<
    {
      violation_type: ViolationType
      base_amount: number
      logic_type: LogicType
      time_window: number
      multiplier: number
    }[]
  >([])
  const [publishing, setPublishing] = useState(false)
  const [publishMessage, setPublishMessage] = useState<string | null>(null)

  // --- Violation Form State ---
  const [licensePlate, setLicensePlate] = useState('')
  const [violationType, setViolationType] = useState<ViolationType>('EXPIRED_METER')
  const [location, setLocation] = useState('')
  const [timestamp, setTimestamp] = useState('')
  const [photoUrl, setPhotoUrl] = useState('')
  const [submittingViolation, setSubmittingViolation] = useState(false)
  const [violationError, setViolationError] = useState<string | null>(null)

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    const reader = new FileReader()
    reader.onloadend = () => {
      if (typeof reader.result === 'string') {
        setPhotoUrl(reader.result)
      }
    }
    reader.readAsDataURL(file)
  }
  
  // --- Submission Result State ---
  const [createdInvoice, setCreatedInvoice] = useState<{
    violation: Violation
    invoice: Invoice
  } | null>(null)

  // Load active rules on mount
  const fetchRules = async () => {
    try {
      setRulesLoading(true)
      setRulesError(null)
      const res = await getActiveFineRule()
      setActiveRule(res.data)
      
      // Initialize editor state with loaded values
      if (res.data?.details) {
        setEditorDetails(
          res.data.details.map((d) => ({
            violation_type: d.violation_type,
            base_amount: d.base_amount,
            logic_type: d.logic_type,
            time_window: d.time_window || 0,
            multiplier: d.multiplier || 1.0,
          }))
        )
      }
    } catch (err: any) {
      setRulesError(err.message || 'Failed to fetch active fine rules.')
    } finally {
      setRulesLoading(false)
    }
  }

  useEffect(() => {
    fetchRules()
    // Pre-populate timestamp with current local ISO string
    const now = new Date()
    // Format to YYYY-MM-DDTHH:MM
    const tzOffset = now.getTimezoneOffset() * 60000
    const localISOTime = new Date(now.getTime() - tzOffset).toISOString().slice(0, 16)
    setTimestamp(localISOTime)
  }, [])

  // Handle fine rule updates
  const handleDetailChange = (
    index: number,
    field: 'base_amount' | 'time_window' | 'multiplier',
    value: number
  ) => {
    const updated = [...editorDetails]
    updated[index] = {
      ...updated[index],
      [field]: value,
    }
    setEditorDetails(updated)
  }

  const handlePublishRules = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      setPublishing(true)
      setPublishMessage(null)
      const payload = {
        details: editorDetails.map((d) => ({
          violation_type: d.violation_type,
          base_amount: Number(d.base_amount),
          logic_type: d.logic_type,
          time_window: Number(d.time_window),
          multiplier: Number(d.multiplier),
        })),
      }
      const res = await publishFineRule(payload)
      setPublishMessage(`Successfully published ruleset version ${res.data.version}!`)
      fetchRules() // refresh view
    } catch (err: any) {
      setPublishMessage(`Error: ${err.message}`)
    } finally {
      setPublishing(false)
    }
  }

  // Handle violation recording
  const handleRecordViolation = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!licensePlate.trim() || !location.trim() || !timestamp) {
      setViolationError('Please fill out all required fields.')
      return
    }

    try {
      setSubmittingViolation(true)
      setViolationError(null)
      setCreatedInvoice(null)

      // Convert local datetime-local picker value to full ISO UTC string
      const utcTimestamp = new Date(timestamp).toISOString()

      const payload = {
        license_plate: licensePlate.toUpperCase().trim(),
        violation_type: violationType,
        location: location.trim(),
        timestamp: utcTimestamp,
        photo_url: photoUrl.trim() || undefined,
      }

      const res = await createViolation(payload)
      
      // Wait a moment for billing async calculation to complete, then poll/fetch invoice
      // For a better UX, wait 1.5 seconds so billing processor has stored the invoice
      await new Promise((resolve) => setTimeout(resolve, 1500))

      // Now query invoices by plate to find the newest invoice generated
      // Wait, can fetch via the response data of violation if violation service returned invoice ID,
      // but violation service responds with Violation data. So we fetch vehicle invoices.
      const invoicesData = await getVehicleInvoices(payload.license_plate)
      
      if (invoicesData.data && invoicesData.data.length > 0) {
        // Find matching invoice for this violation
        const matched = invoicesData.data.find(
          (inv: Invoice) => inv.violation_id === res.data.id
        )
        if (matched) {
          setCreatedInvoice({
            violation: res.data,
            invoice: matched,
          })
          // Reset form fields
          setLicensePlate('')
          setLocation('')
          setPhotoUrl('')
        } else {
          setViolationError('Violation recorded but invoice calculation is taking longer. Check Member history.')
        }
      } else {
        setViolationError('Violation recorded but no invoice found. Check connection.')
      }

    } catch (err: any) {
      setViolationError(err.message || 'Failed to record violation.')
    } finally {
      setSubmittingViolation(false)
    }
  }

  return (
    <div className="space-y-8 animate-fade-in">
      {/* Header Banner */}
      <div className="bg-gradient-to-r from-blue-700 to-indigo-900 text-white rounded-2xl p-6 md:p-8 shadow-md">
        <h1 className="text-2xl md:text-3xl font-extrabold tracking-tight">Officer Dashboard</h1>
        <p className="mt-2 text-blue-100 max-w-2xl text-sm md:text-base">
          Record public parking violations in real-time, generate invoice fine statements, and publish updated fee formulas instantly.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-8">
        
        {/* ================= COLUMN 1: VIOLATION SUBMISSION (7 cols) ================= */}
        <div className="lg:col-span-7 space-y-6">
          <div className="card">
            <div className="card-header bg-gray-50 flex items-center gap-2">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" className="w-5 h-5 text-blue-600">
                <path fillRule="evenodd" d="M5.47 5.47a.75.75 0 011.06 0L10 8.94l3.47-3.47a.75.75 0 111.06 1.06L11.06 10l3.47 3.47a.75.75 0 11-1.06 1.06L10 11.06l-3.47 3.47a.75.75 0 11-1.06-1.06L8.94 10 5.47 6.53a.75.75 0 010-1.06z" clipRule="evenodd" />
              </svg>
              <h2 className="text-lg font-bold text-gray-900">Record New Violation</h2>
            </div>
            
            <form onSubmit={handleRecordViolation} className="card-body space-y-4">
              {violationError && <div className="alert-error">{violationError}</div>}

              <div>
                <label className="label">License Plate (Required)</label>
                <input
                  type="text"
                  required
                  placeholder="e.g. B 1234 XYZ"
                  value={licensePlate}
                  onChange={(e) => setLicensePlate(e.target.value)}
                  className="input uppercase font-semibold"
                />
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="label">Violation Type</label>
                  <select
                    value={violationType}
                    onChange={(e) => setViolationType(e.target.value as ViolationType)}
                    className="input"
                  >
                    {VIOLATION_TYPES.map((t) => (
                      <option key={t.value} value={t.value}>
                        {t.label}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="label">Violation Timestamp</label>
                  <input
                    type="datetime-local"
                    required
                    value={timestamp}
                    onChange={(e) => setTimestamp(e.target.value)}
                    className="input"
                  />
                </div>
              </div>

              <div>
                <label className="label">Location (Required)</label>
                <input
                  type="text"
                  required
                  placeholder="e.g. Jl. Sudirman Kav 21, Jakarta"
                  value={location}
                  onChange={(e) => setLocation(e.target.value)}
                  className="input"
                />
              </div>

              <div>
                <label className="label text-xs font-bold text-gray-700">Violation Evidence Photo (Optional)</label>
                <div className="space-y-3 mt-1">
                  {/* File Upload Input */}
                  <div className="flex items-center justify-center w-full">
                    <label className="flex flex-col items-center justify-center w-full h-28 border-2 border-gray-300 border-dashed rounded-xl cursor-pointer bg-gray-50 hover:bg-gray-100/70 hover:border-gray-400 transition-all">
                      <div className="flex flex-col items-center justify-center pt-4 pb-4">
                        <svg className="w-8 h-8 mb-2 text-gray-400" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 20 16">
                          <path stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M13 13h3a3 3 0 0 0 0-6h-.025A5.56 5.56 0 0 0 16 6.5 5.5 5.5 0 0 0 5.207 5.021C5.137 5.017 5.071 5 5 5a4 4 0 0 0 0 8h2.167M10 15V6m0 0L8 8m2-2 2 2"/>
                        </svg>
                        <p className="text-xs text-gray-500"><span className="font-semibold text-indigo-600">Click to upload photo</span> or drag & drop</p>
                        <p className="text-[10px] text-gray-400 mt-0.5">Supports PNG, JPG, or JPEG</p>
                      </div>
                      <input
                        type="file"
                        accept="image/*"
                        onChange={handleFileChange}
                        className="hidden"
                      />
                    </label>
                  </div>

                  {/* Or input URL alternative */}
                  <div className="relative flex py-1 items-center">
                    <div className="flex-grow border-t border-gray-200"></div>
                    <span className="flex-shrink mx-3 text-gray-400 text-[10px] font-bold uppercase tracking-wider">Or enter URL</span>
                    <div className="flex-grow border-t border-gray-200"></div>
                  </div>

                  <input
                    type="url"
                    placeholder="e.g. https://images.unsplash.com/photo-..."
                    value={photoUrl.startsWith('data:') ? 'Local Image Selected (Base64 String)' : photoUrl}
                    onChange={(e) => setPhotoUrl(e.target.value)}
                    disabled={photoUrl.startsWith('data:')}
                    className="input text-xs"
                  />

                  {/* Image Preview & Clear Button */}
                  {photoUrl && (
                    <div className="relative mt-2 p-2 border rounded-lg bg-gray-50 flex items-center justify-between gap-4 animate-fade-in">
                      <div className="flex items-center gap-3">
                        <img
                          src={photoUrl}
                          alt="Preview"
                          className="h-10 w-14 object-cover rounded border"
                        />
                        <span className="text-xs font-semibold text-gray-500 truncate max-w-[200px]">
                          {photoUrl.startsWith('data:') ? 'Uploaded local image' : 'External image URL'}
                        </span>
                      </div>
                      <button
                        type="button"
                        onClick={() => setPhotoUrl('')}
                        className="text-xs text-red-500 hover:text-red-700 font-bold px-2 py-1 hover:bg-red-50 rounded"
                      >
                        Clear
                      </button>
                    </div>
                  )}
                </div>
              </div>

              <button
                type="submit"
                disabled={submittingViolation}
                className="w-full btn-primary py-2.5 mt-2"
              >
                {submittingViolation ? (
                  <span className="flex items-center justify-center gap-2">
                    <svg className="animate-spin h-5 w-5 text-white" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                    </svg>
                    Recording & Calculating Fine...
                  </span>
                ) : (
                  'Record Violation & Issue Invoice'
                )}
              </button>
            </form>
          </div>

          {/* Success Invoice Panel */}
          {createdInvoice && (
            <div className="card border-emerald-500 bg-emerald-50/20 overflow-hidden animate-slide-up">
              <div className="card-header bg-emerald-600 text-white flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" className="w-5 h-5">
                    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clipRule="evenodd" />
                  </svg>
                  <span className="font-bold text-sm">Violation Confirmed & Invoice Issued</span>
                </div>
                <span className="text-xs bg-emerald-700 px-2 py-0.5 rounded-full font-mono">
                  ID: {createdInvoice.violation.id}
                </span>
              </div>
              <div className="card-body space-y-4">
                <div className="flex flex-col sm:flex-row sm:justify-between border-b border-emerald-100 pb-3 gap-2">
                  <div>
                    <span className="text-xs text-gray-500 block uppercase font-semibold">Invoice No</span>
                    <span className="font-mono font-bold text-gray-900">{createdInvoice.invoice.invoice_no}</span>
                  </div>
                  <div>
                    <span className="text-xs text-gray-500 block uppercase font-semibold text-sm sm:text-right">Calculated Fine</span>
                    <span className="text-lg font-extrabold text-emerald-700">Rp {createdInvoice.invoice.calculated_amount.toLocaleString('id-ID')}</span>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4 text-xs">
                  <div>
                    <span className="text-gray-500 block uppercase font-semibold">License Plate</span>
                    <span className="font-bold text-gray-950">{createdInvoice.violation.license_plate}</span>
                  </div>
                  <div>
                    <span className="text-gray-500 block uppercase font-semibold">Violation Type</span>
                    <span className="font-semibold text-gray-900">{createdInvoice.violation.violation_type.replace(/_/g, ' ')}</span>
                  </div>
                  <div>
                    <span className="text-gray-500 block uppercase font-semibold">Location</span>
                    <span className="text-gray-900">{createdInvoice.violation.location}</span>
                  </div>
                  <div>
                    <span className="text-gray-500 block uppercase font-semibold">Timestamp</span>
                    <span className="text-gray-900">{new Date(createdInvoice.violation.timestamp).toLocaleString()}</span>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* ================= COLUMN 2: RULES CONFIGURATION (5 cols) ================= */}
        <div className="lg:col-span-5 space-y-6">
          <div className="card">
            <div className="card-header bg-gray-50 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" className="w-5 h-5 text-indigo-600">
                  <path fillRule="evenodd" d="M11.47 3.03a.75.75 0 011.06 0l3 3a.75.75 0 010 1.06l-3 3a.75.75 0 11-1.06-1.06L13.19 8H10a4 4 0 00-4 4v1a.75.75 0 01-1.5 0v-1a5.5 5.5 0 015.5-5.5h3.19l-1.72-1.72a.75.75 0 010-1.06z" clipRule="evenodd" />
                </svg>
                <h2 className="text-lg font-bold text-gray-900">Fine Formula Ruleset</h2>
              </div>
              {activeRule && (
                <span className="text-xs bg-indigo-100 text-indigo-800 font-bold px-2 py-0.5 rounded-full">
                  Version {activeRule.version}
                </span>
              )}
            </div>

            <div className="card-body">
              {rulesLoading ? (
                <div className="flex justify-center py-8">
                  <svg className="animate-spin h-6 w-6 text-indigo-600" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                </div>
              ) : rulesError ? (
                <div className="alert-error">{rulesError}</div>
              ) : (
                <form onSubmit={handlePublishRules} className="space-y-5">
                  {publishMessage && (
                    <div
                      className={`text-xs p-3 rounded-lg border ${
                        publishMessage.startsWith('Error')
                          ? 'bg-red-50 border-red-200 text-red-700'
                          : 'bg-emerald-50 border-emerald-200 text-emerald-700'
                      }`}
                    >
                      {publishMessage}
                    </div>
                  )}

                  {editorDetails.map((detail, index) => (
                    <div key={detail.violation_type} className="p-3 bg-gray-50 rounded-lg border border-gray-200 space-y-3">
                      <span className="text-xs font-bold text-gray-700 uppercase block tracking-wider">
                        {detail.violation_type.replace(/_/g, ' ')}
                      </span>
                      
                      <div className="grid grid-cols-2 gap-3">
                        <div>
                          <label className="text-[11px] font-semibold text-gray-500 block mb-0.5">Base Amount (IDR)</label>
                          <input
                            type="number"
                            min="1000"
                            required
                            value={detail.base_amount}
                            onChange={(e) => handleDetailChange(index, 'base_amount', Number(e.target.value))}
                            className="input text-xs"
                          />
                        </div>
                        <div>
                          <label className="text-[11px] font-semibold text-gray-500 block mb-0.5">Repeat Multiplier (2+)</label>
                          <input
                            type="number"
                            step="0.1"
                            min="1.0"
                            required
                            value={detail.multiplier}
                            onChange={(e) => handleDetailChange(index, 'multiplier', Number(e.target.value))}
                            className="input text-xs"
                          />
                        </div>
                      </div>
                    </div>
                  ))}

                  <div className="bg-amber-50 border border-amber-200 rounded-lg p-3 text-[11px] text-amber-800 space-y-1">
                    <p className="font-bold">⚠️ Caution on Publishing</p>
                    <p>
                      Publishing a new ruleset increments the formula version. Existing unpaid invoices maintain their original version locking; only new violations are calculated with this updated config.
                    </p>
                  </div>

                  <button
                    type="submit"
                    disabled={publishing}
                    className="w-full btn-success text-xs font-semibold py-2"
                  >
                    {publishing ? 'Publishing Ruleset...' : 'Publish New Ruleset (v' + ((activeRule?.version || 0) + 1) + ')'}
                  </button>
                </form>
              )}
            </div>
          </div>
        </div>

      </div>
    </div>
  )
}
