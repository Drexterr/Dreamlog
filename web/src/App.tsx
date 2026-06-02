import React from 'react';
import { AuthProvider, useAuth } from './context/AuthContext';
import { Login } from './views/Login';
import { Dashboard } from './views/Dashboard';

const AppContent: React.FC = () => {
  const { isAuthenticated, loading } = useAuth();

  if (loading) {
    return (
      <div style={styles.loadingContainer}>
        <div style={styles.spinner} />
        <p style={{ marginTop: '16px', color: '#cbd5e1', letterSpacing: '1px' }}>VERIFYING SESSION...</p>
      </div>
    );
  }

  return isAuthenticated ? <Dashboard /> : <Login />;
};

export default function App() {
  return (
    <AuthProvider>
      <AppContent />
    </AuthProvider>
  );
}

const styles: Record<string, React.CSSProperties> = {
  loadingContainer: {
    width: '100vw',
    height: '100vh',
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    background: '#0b1329',
    fontFamily: "'Inter', sans-serif",
  },
  spinner: {
    width: '40px',
    height: '40px',
    border: '3px solid rgba(255, 255, 255, 0.1)',
    borderTopColor: '#00b4d8',
    borderRadius: '50%',
    animation: 'spin 1s infinite linear',
  },
};
