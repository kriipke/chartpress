import React from 'react';
import ReactDOM from 'react-dom/client';
import Wizard from './App';
import './App.css'; // Include the custom CSS

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <React.StrictMode>
    <Wizard />
  </React.StrictMode>
);
