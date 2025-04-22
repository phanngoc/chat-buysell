import '../styles/globals.css';
import { useState, useEffect } from 'react';
import { AuthProvider } from '../components/AuthContext';

function MyApp({ Component, pageProps }) {
  // Add any global state or providers here
  
  return (
    <AuthProvider>
      <Component {...pageProps} />
    </AuthProvider>
  );
}

export default MyApp;