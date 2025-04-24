import React from 'react'
import ReactDOM from 'react-dom/client' // Ensure correct import for React 18
import App from './App'
import ErrorBoundary from './ErrorBoundary'

const root = ReactDOM.createRoot(document.getElementById('root')) // Use createRoot
root.render(
  <React.StrictMode>
    <ErrorBoundary>
      <App />
    </ErrorBoundary>
  </React.StrictMode>
)

