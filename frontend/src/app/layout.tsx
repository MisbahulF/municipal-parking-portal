import type { Metadata } from 'next'
import './globals.css'
import RoleSwitcher from '@/components/RoleSwitcher'

export const metadata: Metadata = {
  title: {
    default: 'Parking Violation Portal',
    template: '%s | Parking Violation Portal',
  },
  description:
    'A unified portal for managing parking violations, invoices, and payments.',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en" className="h-full">
      <body className="h-full flex flex-col">
        {/* ── Top Navigation ─────────────────────────────────────────────── */}
        <header className="sticky top-0 z-40 border-b border-gray-200 bg-white shadow-sm">
          <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
            <div className="flex items-center gap-3">
              {/* Logo icon */}
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-blue-600 text-white">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  viewBox="0 0 24 24"
                  fill="currentColor"
                  className="h-5 w-5"
                >
                  <path d="M3.375 4.5C2.339 4.5 1.5 5.34 1.5 6.375V13.5h12V6.375c0-1.036-.84-1.875-1.875-1.875h-8.25zM13.5 15h-12v2.625c0 1.035.84 1.875 1.875 1.875H5.25a3.375 3.375 0 006.75 0h2.625v-4.5z" />
                  <path d="M8.25 19.5a1.5 1.5 0 10-3 0 1.5 1.5 0 003 0zM15.75 6.75a.75.75 0 00-.75.75v11.25c0 .087.015.17.042.248a3.375 3.375 0 015.958.464c.853-.175 1.522-.935 1.464-1.883a18.659 18.659 0 00-3.732-10.104 1.837 1.837 0 00-1.47-.725H15.75z" />
                  <path d="M19.5 19.5a1.5 1.5 0 10-3 0 1.5 1.5 0 003 0z" />
                </svg>
              </div>
              <div>
                <p className="text-sm font-bold text-gray-900 leading-tight">
                  Parking Violation
                </p>
                <p className="text-xs text-gray-500 leading-tight">Portal</p>
              </div>
            </div>

            {/* Nav links / Role switcher */}
            <RoleSwitcher />
          </div>
        </header>

        {/* ── Page Content ─────────────────────────────────────────────────── */}
        <main className="flex-1 mx-auto w-full max-w-7xl px-4 sm:px-6 lg:px-8 py-8">
          {children}
        </main>

        {/* ── Footer ───────────────────────────────────────────────────────── */}
        <footer className="border-t border-gray-200 bg-white">
          <div className="mx-auto max-w-7xl px-4 py-4 sm:px-6 lg:px-8">
            <p className="text-center text-xs text-gray-400">
              Parking Violation Portal — API Gateway:{' '}
              <code className="font-mono">{process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'}</code>
            </p>
          </div>
        </footer>
      </body>
    </html>
  )
}
