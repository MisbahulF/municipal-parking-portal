'use client'

import React from 'react'
import { usePathname, useRouter } from 'next/navigation'

export default function RoleSwitcher() {
  const pathname = usePathname()
  const router = useRouter()

  // Determine current active role based on path
  const isOfficer = pathname.startsWith('/officer')
  const isMember = pathname.startsWith('/member')
  const currentRole = isOfficer ? 'Officer' : isMember ? 'Member' : 'Guest'

  const handleRoleChange = (role: 'officer' | 'member' | 'guest') => {
    if (role === 'officer') {
      router.push('/officer')
    } else if (role === 'member') {
      router.push('/member')
    } else {
      router.push('/')
    }
  }

  return (
    <div className="flex items-center gap-4 bg-gray-100 p-1.5 rounded-xl border border-gray-200">
      {/* Indicator */}
      <span className="hidden sm:inline-block text-xs font-semibold uppercase tracking-wider text-gray-500 px-2">
        Active Role:
      </span>

      <div className="flex items-center gap-1 bg-white p-1 rounded-lg shadow-sm border border-gray-100">
        <button
          onClick={() => handleRoleChange('officer')}
          className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-semibold transition-all ${
            isOfficer
              ? 'bg-blue-600 text-white shadow-sm'
              : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
          }`}
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 20 20"
            fill="currentColor"
            className="w-3.5 h-3.5"
          >
            <path
              fillRule="evenodd"
              d="M10 2a1 1 0 00-1 1v1a1 1 0 002 0V3a1 1 0 00-1-1zM4 4h3a3 3 0 006 0h3a2 2 0 012 2v9a2 2 0 01-2 2H4a2 2 0 01-2-2V6a2 2 0 012-2zm2.5 7a1.5 1.5 0 100-3 1.5 1.5 0 000 3zm2.45-1.45a.75.75 0 011.06 0l1.5 1.5a.75.75 0 11-1.06 1.06l-.97-.97V15a.75.75 0 01-1.5 0v-2.91l-.97.97a.75.75 0 11-1.06-1.06l1.5-1.5z"
              clipRule="evenodd"
            />
          </svg>
          Officer Portal
        </button>

        <button
          onClick={() => handleRoleChange('member')}
          className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-semibold transition-all ${
            isMember
              ? 'bg-emerald-600 text-white shadow-sm'
              : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
          }`}
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 20 20"
            fill="currentColor"
            className="w-3.5 h-3.5"
          >
            <path d="M10 8a3 3 0 100-6 3 3 0 000 6zM3.465 14.493a1.23 1.23 0 00.41 1.412A9.957 9.957 0 0010 18c2.31 0 4.438-.784 6.131-2.1.43-.333.604-.903.408-1.41a7.002 7.002 0 00-13.074.003z" />
          </svg>
          Member Portal
        </button>

        <button
          onClick={() => handleRoleChange('guest')}
          className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-semibold transition-all ${
            currentRole === 'Guest'
              ? 'bg-gray-800 text-white shadow-sm'
              : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
          }`}
        >
          Home
        </button>
      </div>
    </div>
  )
}
