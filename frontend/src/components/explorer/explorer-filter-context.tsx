'use client'

import { createContext, useContext, useMemo, useState, type ReactNode } from 'react'

interface ExplorerFilterContextValue {
  showPublications: boolean
  showUsers: boolean
  setFilters: (filters: { publications: boolean; users: boolean }) => void
  setSubmittedSearchActive: (active: boolean) => void
  submittedSearchActive: boolean
  togglePublications: () => void
  toggleUsers: () => void
}

const ExplorerFilterContext = createContext<ExplorerFilterContextValue | null>(null)

export function ExplorerFilterProvider({ children }: { children: ReactNode }) {
  const [submittedSearchActive, setSubmittedSearchActive] = useState(false)
  const [showPublications, setShowPublications] = useState(true)
  const [showUsers, setShowUsers] = useState(false)

  const value = useMemo<ExplorerFilterContextValue>(
    () => ({
      showPublications,
      showUsers,
      setFilters: (filters) => {
        if (!filters.publications && !filters.users) return
        setShowPublications(filters.publications)
        setShowUsers(filters.users)
      },
      submittedSearchActive,
      setSubmittedSearchActive,
      togglePublications: () => {
        setShowPublications((current) => {
          if (current && !showUsers) return current
          return !current
        })
      },
      toggleUsers: () => {
        setShowUsers((current) => {
          if (current && !showPublications) return current
          return !current
        })
      },
    }),
    [showPublications, showUsers, submittedSearchActive],
  )

  return (
    <ExplorerFilterContext.Provider value={value}>
      {children}
    </ExplorerFilterContext.Provider>
  )
}

export function useExplorerFilters() {
  const value = useContext(ExplorerFilterContext)
  if (!value) throw new Error('useExplorerFilters must be used within ExplorerFilterProvider')
  return value
}
