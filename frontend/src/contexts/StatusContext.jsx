import React, { createContext, useContext, useEffect, useState } from 'react'
import api from '../services/api'

const StatusContext = createContext()

export const StatusProvider = ({ children }) => {
  const [serverDown, setServerDown] = useState('')

  useEffect(() => {
    const interceptor = api.interceptors.response.use(
      (response) => {
        setServerDown('')
        return response
      },
      (error) => {
        if (!error.response) {
          setServerDown('Not connected to server.')
        }
        return Promise.reject(error)
      }
    )
    return () => {
      api.interceptors.response.eject(interceptor)
    }
  }, [])

  return (
    <StatusContext.Provider value={{ serverDown, setServerDown }}>
      {children}
    </StatusContext.Provider>
  )
}

export const useStatus = () => {
  const ctx = useContext(StatusContext)
  if (!ctx) throw new Error('useStatus must be used within a StatusProvider')
  return ctx
}
