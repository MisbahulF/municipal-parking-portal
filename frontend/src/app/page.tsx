'use client'

import React, { useEffect, useState } from 'react'
import Link from 'next/link'
import { getAllInvoices } from '@/lib/api'
import type { Invoice } from '@/lib/types'

export default function HomePage() {
  const [invoices, setInvoices] = useState<Invoice[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchTransactions = async () => {
      try {
        setLoading(true)
        setError(null)
        const res = await getAllInvoices()
        setInvoices(res.data || [])
      } catch (err: any) {
        setError(err.message || 'Failed to fetch global transaction history.')
      } finally {
        setLoading(false)
      }
    }
    fetchTransactions()
  }, [])

  return (
    <div className="space-y-10 animate-fade-in">
      {/* Hero Welcome Banner */}
      <div className="relative overflow-hidden bg-gradient-to-br from-slate-900 via-indigo-950 to-blue-900 text-white rounded-3xl p-8 md:p-12 shadow-xl border border-indigo-900/50">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(99,102,241,0.15),transparent)] pointer-events-none" />
        
        <div className="relative z-10 max-w-3xl space-y-4">
          <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-blue-500/20 text-blue-300 text-xs font-semibold uppercase tracking-wider">
            🏛️ Smart City Initiative
          </span>
          <h1 className="text-3xl md:text-5xl font-black tracking-tight leading-tight">
            Integrated Parking <br className="hidden sm:inline" />
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-blue-400 to-indigo-300">
              Violation Portal
            </span>
          </h1>
          <p className="text-slate-300 text-sm md:text-base leading-relaxed max-w-xl">
            A state-of-the-art municipal gateway for tracking parking offenses, calculating fine formulas using versioned rulesets, and processing instant member settlements.
          </p>
        </div>
      </div>

      {/* Role Navigation Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Officer Card */}
        <Link
          href="/officer"
          className="group card p-6 hover:border-blue-500 hover:shadow-lg transition-all duration-300 cursor-pointer flex flex-col justify-between"
        >
          <div className="space-y-4">
            <div className="h-12 w-12 rounded-xl bg-blue-50 flex items-center justify-center text-blue-600 group-hover:bg-blue-600 group-hover:text-white transition-colors duration-300 shadow-sm">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth="1.5" stroke="currentColor" className="w-6 h-6">
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m0-10.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.75c0 5.592 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.57-.598-3.75h-.152c-3.196 0-6.1-1.249-8.25-3.286zm0 13.036h.008v.008H12v-.008z" />
              </svg>
            </div>
            <div>
              <h3 className="font-extrabold text-gray-900 group-hover:text-blue-600 transition-colors">Officer Portal</h3>
              <p className="text-xs text-gray-500 mt-1 leading-relaxed">
                Log violations, upload offense logs, view rule version controls, and publish new base rates.
              </p>
            </div>
          </div>
          <span className="text-xs font-semibold text-blue-600 mt-6 inline-flex items-center gap-1 group-hover:translate-x-1 transition-transform">
            Enter Officer Portal &rarr;
          </span>
        </Link>

        {/* Member Card */}
        <Link
          href="/member"
          className="group card p-6 hover:border-emerald-500 hover:shadow-lg transition-all duration-300 cursor-pointer flex flex-col justify-between"
        >
          <div className="space-y-4">
            <div className="h-12 w-12 rounded-xl bg-emerald-50 flex items-center justify-center text-emerald-600 group-hover:bg-emerald-600 group-hover:text-white transition-colors duration-300 shadow-sm">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth="1.5" stroke="currentColor" className="w-6 h-6">
                <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 8.25h19.5M2.25 9h19.5m-16.5 5.25h6m-6 2.25h3m-3.75 3h15a2.25 2.25 0 002.25-2.25V6.75A2.25 2.25 0 0019.5 4.5h-15a2.25 2.25 0 00-2.25 2.25v10.5A2.25 2.25 0 004.5 19.5z" />
              </svg>
            </div>
            <div>
              <h3 className="font-extrabold text-gray-900 group-hover:text-emerald-600 transition-colors">Member Portal</h3>
              <p className="text-xs text-gray-500 mt-1 leading-relaxed">
                Search ticket histories by vehicle license plate, view billing details, and make mock-gateway fine payments.
              </p>
            </div>
          </div>
          <span className="text-xs font-semibold text-emerald-600 mt-6 inline-flex items-center gap-1 group-hover:translate-x-1 transition-transform">
            Enter Member Portal &rarr;
          </span>
        </Link>
      </div>

      {/* master Transaction History table */}
      <div className="card">
        <div className="card-header bg-gray-50 flex items-center justify-between">
          <h2 className="text-sm font-bold text-gray-800 uppercase tracking-wider">
            Municipal Transaction Registry (Flow 5)
          </h2>
          <button
            onClick={() => {
              setLoading(true)
              getAllInvoices().then((res) => {
                setInvoices(res.data || [])
                setLoading(false)
              })
            }}
            className="text-xs font-semibold text-blue-600 hover:text-blue-800 flex items-center gap-1 transition-colors"
          >
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor" className="w-3.5 h-3.5">
              <path strokeLinecap="round" strokeLinejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99" />
            </svg>
            Refresh
          </button>
        </div>

        <div className="overflow-x-auto">
          {loading ? (
            <div className="flex justify-center py-12">
              <svg className="animate-spin h-8 w-8 text-indigo-600" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
            </div>
          ) : error ? (
            <div className="p-6 text-xs text-red-600 text-center">{error}</div>
          ) : invoices.length === 0 ? (
            <div className="text-center py-12 text-xs text-gray-400">
              No transactions recorded in the system yet.
            </div>
          ) : (
            <table className="min-w-full divide-y divide-gray-200 text-left text-xs">
              <thead className="bg-gray-50 text-gray-500 font-semibold uppercase tracking-wider">
                <tr>
                  <th className="px-6 py-3">Invoice No</th>
                  <th className="px-6 py-3">License Plate</th>
                  <th className="px-6 py-3">Violation Type</th>
                  <th className="px-6 py-3">Location</th>
                  <th className="px-6 py-3">Date</th>
                  <th className="px-6 py-3">Rule Version</th>
                  <th className="px-6 py-3 text-right">Fine Amount</th>
                  <th className="px-6 py-3 text-center">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 bg-white text-gray-700">
                {invoices.map((inv) => (
                  <tr key={inv.id} className="hover:bg-gray-50 transition-colors">
                    <td className="px-6 py-4 font-mono font-bold text-gray-900">{inv.invoice_no}</td>
                    <td className="px-6 py-4 font-semibold uppercase text-gray-950">{inv.violation?.license_plate}</td>
                    <td className="px-6 py-4">{inv.violation?.violation_type?.replace(/_/g, ' ')}</td>
                    <td className="px-6 py-4 max-w-[150px] truncate" title={inv.violation?.location}>
                      {inv.violation?.location}
                    </td>
                    <td className="px-6 py-4">
                      {inv.violation?.timestamp ? new Date(inv.violation.timestamp).toLocaleString() : 'N/A'}
                    </td>
                    <td className="px-6 py-4 text-center">
                      <span className="inline-flex items-center px-2 py-0.5 rounded bg-slate-100 text-slate-800 font-bold font-mono">
                        v{inv.applied_fine_rule?.version || 1}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-right font-extrabold text-gray-900">
                      Rp {inv.calculated_amount.toLocaleString('id-ID')}
                    </td>
                    <td className="px-6 py-4 text-center">
                      <span
                        className={
                          inv.status === 'PAID'
                            ? 'badge-paid font-semibold'
                            : inv.status === 'VOIDED'
                            ? 'badge-voided font-semibold'
                            : 'badge-unpaid font-semibold'
                        }
                      >
                        {inv.status}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  )
}
