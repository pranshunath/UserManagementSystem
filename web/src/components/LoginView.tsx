import React, { useState } from 'react';
import { useAuth } from '../context/AuthContext';
import { ShieldCheck, Lock, Mail, AlertCircle, ArrowRight, Sparkles } from 'lucide-react';
import { ApiClientError } from '../api/client';

export const LoginView: React.FC = () => {
  const { login } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [requestId, setRequestId] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email || !password) {
      setError('Please enter both email and password');
      return;
    }

    setError(null);
    setRequestId(null);
    setIsSubmitting(true);

    try {
      await login(email, password);
    } catch (err: unknown) {
      if (err instanceof ApiClientError) {
        setError(err.message);
        if (err.requestId) setRequestId(err.requestId);
      } else {
        setError('An unexpected error occurred. Please try again.');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const fillDemo = (demoEmail: string, demoPass: string) => {
    setEmail(demoEmail);
    setPassword(demoPass);
    setError(null);
    setRequestId(null);
  };

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '24px',
    }}>
      <div className="glass-panel fade-in" style={{
        width: '100%',
        maxWidth: '440px',
        padding: '36px',
        position: 'relative',
        overflow: 'hidden',
      }}>
        {/* Luminous Top Bar Accent */}
        <div style={{
          position: 'absolute',
          top: 0,
          left: 0,
          right: 0,
          height: '3px',
          background: 'linear-gradient(90deg, #6366f1, #06b6d4, #a855f7)',
        }} />

        {/* Brand Header */}
        <div style={{ textAlign: 'center', marginBottom: '28px' }}>
          <div style={{
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
            width: '56px',
            height: '56px',
            borderRadius: '16px',
            background: 'linear-gradient(135deg, rgba(99, 102, 241, 0.2), rgba(6, 182, 212, 0.1))',
            border: '1px solid rgba(99, 102, 241, 0.3)',
            color: '#818cf8',
            marginBottom: '16px',
          }}>
            <ShieldCheck size={30} />
          </div>
          <h1 style={{ fontSize: '1.75rem', fontWeight: 700, marginBottom: '6px' }}>Nexus Core</h1>
          <p style={{ color: 'var(--text-secondary)', fontSize: '0.875rem' }}>
            Enterprise User Management System
          </p>
        </div>

        {/* Error Alert Box with Request ID */}
        {error && (
          <div className="fade-in" style={{
            background: 'var(--rose-bg)',
            border: '1px solid var(--rose-border)',
            borderRadius: 'var(--radius-md)',
            padding: '12px 14px',
            marginBottom: '20px',
            display: 'flex',
            flexDirection: 'column',
            gap: '4px',
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', color: 'var(--rose)' }}>
              <AlertCircle size={18} />
              <span style={{ fontSize: '0.875rem', fontWeight: 500 }}>{error}</span>
            </div>
            {requestId && (
              <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)', fontFamily: 'monospace', marginLeft: '26px' }}>
                Trace ID: {requestId}
              </span>
            )}
          </div>
        )}

        {/* Login Form */}
        <form onSubmit={handleSubmit}>
          <div className="input-group">
            <label className="input-label" htmlFor="email">Work Email</label>
            <div style={{ position: 'relative' }}>
              <Mail size={16} style={{ position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)' }} />
              <input
                id="email"
                type="email"
                className="input-field"
                placeholder="name@company.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                style={{ paddingLeft: '38px' }}
                autoComplete="email"
                required
              />
            </div>
          </div>

          <div className="input-group">
            <label className="input-label" htmlFor="password">Password</label>
            <div style={{ position: 'relative' }}>
              <Lock size={16} style={{ position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)' }} />
              <input
                id="password"
                type="password"
                className="input-field"
                placeholder="••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                style={{ paddingLeft: '38px' }}
                autoComplete="current-password"
                required
              />
            </div>
          </div>

          <button
            type="submit"
            className="btn btn-primary"
            style={{ width: '100%', padding: '12px', fontSize: '0.95rem', marginTop: '8px' }}
            disabled={isSubmitting}
          >
            {isSubmitting ? (
              <span>Authenticating...</span>
            ) : (
              <>
                <span>Sign in to Workspace</span>
                <ArrowRight size={16} />
              </>
            )}
          </button>
        </form>

        {/* Quick Demo Test Presets */}
        <div style={{ marginTop: '28px', paddingTop: '20px', borderTop: '1px solid var(--border-subtle)' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '6px', marginBottom: '10px', color: 'var(--text-muted)', fontSize: '0.75rem', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            <Sparkles size={14} color="#818cf8" />
            <span>Quick-fill Test Accounts</span>
          </div>
          <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
            <button
              type="button"
              className="btn btn-secondary"
              style={{ fontSize: '0.75rem', padding: '6px 12px' }}
              onClick={() => fillDemo('admin@system.io', 'adminpassword123')}
            >
              👑 Admin
            </button>
            <button
              type="button"
              className="btn btn-secondary"
              style={{ fontSize: '0.75rem', padding: '6px 12px' }}
              onClick={() => fillDemo('staff@system.io', 'staffpassword123')}
            >
              👤 Staff / Employee
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
