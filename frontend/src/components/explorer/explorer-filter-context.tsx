'use client'

import { createContext, useContext, useMemo, useState, type ReactNode } from 'react'

interface ExplorerFilterContextValue {
  showPublications: boolean
  showUsers: boolean
  setFilters: (filters: { publications: boolean; users: boolean }) => void
  setSubmittedSearchActive: (active: boolean) => void
  submittedSearchActive: boolean
  /** Bascule « Publications ». Renvoie false si refusé (dernier filtre actif). */
  tryTogglePublications: () => boolean
  /** Bascule « Utilisateurs ». Renvoie false si refusé (dernier filtre actif). */
  tryToggleUsers: () => boolean
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
      tryTogglePublications: () => {
        if (showPublications && !showUsers) return false
        setShowPublications((current) => !current)
        return true
      },
      tryToggleUsers: () => {
        if (showUsers && !showPublications) return false
        setShowUsers((current) => !current)
        return true
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
