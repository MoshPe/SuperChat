import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import App from './App.jsx'
import './index.css'
import { StatusProvider } from './contexts/StatusContext.jsx'
import { ThemeProvider } from './contexts/ThemeContext.jsx'

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <ThemeProvider>
      <StatusProvider>
        <BrowserRouter>
          <App />
        </BrowserRouter>
      </StatusProvider>
    </ThemeProvider>
  </React.StrictMode>,
)
