import React, { useState, useEffect } from 'react';
import { AuthProvider, useAuth } from './context/AuthContext';
import { LoginView } from './components/LoginView';
import { DashboardView } from './components/DashboardView';
import UsersView from './components/UsersView';
import { DepartmentsView } from './components/DepartmentsView';
import RolesView from './components/RolesView';
import { systemApi, userApi, departmentApi, roleApi } from './api/client';
import type { User, Department, Role } from './api/types';
import {
  Users,
  Building2,
  ShieldAlert,
  LayoutDashboard,
  LogOut,
  RefreshCw,
} from 'lucide-react';

type Tab = 'dashboard' | 'users' | 'departments' | 'roles';

const MainLayout: React.FC = () => {
  const { user, roles, logout, isAdmin } = useAuth();
  const [activeTab, setActiveTab] = useState<Tab>('dashboard');
  const [isOnline, setIsOnline] = useState<boolean | null>(null);

  // Quick live summary counts for dashboard
  const [userCount, setUserCount] = useState<number>(0);
  const [usersList, setUsersList] = useState<User[]>([]);
  const [deptsList, setDeptsList] = useState<Department[]>([]);
  const [rolesList, setRolesList] = useState<Role[]>([]);
  const [isLoadingData, setIsLoadingData] = useState(false);

  const fetchLiveCounts = async () => {
    setIsLoadingData(true);
    try {
      const [uRes, dRes, rRes] = await Promise.all([
        userApi.list({ limit: 100 }),
        departmentApi.list(),
        roleApi.list(),
      ]);
      setUserCount(uRes.pagination?.total_count ?? uRes.users.length);
      setUsersList(uRes.users);
      setDeptsList(dRes);
      setRolesList(rRes);
    } catch (err) {
      console.error('Failed to load initial workspace data:', err);
    } finally {
      setIsLoadingData(false);
    }
  };

  // Poll system health every 10 seconds
  useEffect(() => {
    const checkHealth = async () => {
      const ok = await systemApi.checkHealth();
      setIsOnline(ok);
    };
    checkHealth();
    const interval = setInterval(checkHealth, 10000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    fetchLiveCounts();
  }, []);

  return (
    <div style={{ display: 'flex', minHeight: '100vh' }}>
      {/* 1. Sleek Navigation Sidebar */}
      <aside style={{
        width: '260px',
        backgroundColor: 'var(--bg-surface)',
        borderRight: '1px solid var(--border-subtle)',
        display: 'flex',
        flexDirection: 'column',
        padding: '24px 16px',
        flexShrink: 0,
      }}>
        {/* Brand */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px', padding: '0 8px', marginBottom: '32px' }}>
          <div style={{
            width: '38px',
            height: '38px',
            borderRadius: '10px',
            background: 'linear-gradient(135deg, #6366f1, #06b6d4)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: '#fff',
            fontWeight: 800,
            fontSize: '1.1rem',
            boxShadow: '0 4px 12px var(--primary-glow)',
          }}>
            N
          </div>
          <div>
            <div style={{ fontFamily: 'var(--font-heading)', fontWeight: 700, fontSize: '1.125rem' }}>Nexus Core</div>
            <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>Enterprise Platform</div>
          </div>
        </div>

        {/* Navigation Items */}
        <nav style={{ display: 'flex', flexDirection: 'column', gap: '4px', flex: 1 }}>
          <button
            className={`btn ${activeTab === 'dashboard' ? 'btn-primary' : 'btn-ghost'}`}
            style={{ justifyContent: 'flex-start', padding: '10px 14px' }}
            onClick={() => setActiveTab('dashboard')}
          >
            <LayoutDashboard size={18} />
            <span>Dashboard</span>
          </button>

          <button
            className={`btn ${activeTab === 'users' ? 'btn-primary' : 'btn-ghost'}`}
            style={{ justifyContent: 'flex-start', padding: '10px 14px' }}
            onClick={() => setActiveTab('users')}
          >
            <Users size={18} />
            <span>Users Directory</span>
          </button>

          <button
            className={`btn ${activeTab === 'departments' ? 'btn-primary' : 'btn-ghost'}`}
            style={{ justifyContent: 'flex-start', padding: '10px 14px' }}
            onClick={() => setActiveTab('departments')}
          >
            <Building2 size={18} />
            <span>Departments</span>
          </button>

          <button
            className={`btn ${activeTab === 'roles' ? 'btn-primary' : 'btn-ghost'}`}
            style={{ justifyContent: 'flex-start', padding: '10px 14px' }}
            onClick={() => setActiveTab('roles')}
          >
            <ShieldAlert size={18} />
            <span>Roles & RBAC</span>
          </button>
        </nav>

        {/* Microservice & Gateway Health Monitor Widget */}
        <div className="glass-panel" style={{ padding: '14px', marginBottom: '16px' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '8px' }}>
            <span style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', fontWeight: 600 }}>GATEWAY STATUS</span>
            <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
              <div className={isOnline ? 'pulse-dot' : ''} style={{
                width: '8px',
                height: '8px',
                borderRadius: '50%',
                backgroundColor: isOnline ? 'var(--emerald)' : 'var(--rose)',
              }} />
              <span style={{ fontSize: '0.75rem', color: isOnline ? 'var(--emerald)' : 'var(--rose)', fontWeight: 600 }}>
                {isOnline === null ? 'Pinging...' : isOnline ? 'Online :8080' : 'Offline'}
              </span>
            </div>
          </div>
          <div style={{ fontSize: '0.7rem', color: 'var(--text-muted)' }}>
            gRPC Microservice: <span style={{ color: '#818cf8', fontWeight: 500 }}>:50051</span>
          </div>
        </div>

        {/* User profile & Logout */}
        <div style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '12px',
          background: 'rgba(255, 255, 255, 0.03)',
          borderRadius: 'var(--radius-md)',
          border: '1px solid var(--border-subtle)',
        }}>
          <div style={{ overflow: 'hidden', marginRight: '8px' }}>
            <div style={{ fontSize: '0.8125rem', fontWeight: 600, whiteSpace: 'nowrap', textOverflow: 'ellipsis', overflow: 'hidden' }}>
              {user?.name}
            </div>
            <div style={{ fontSize: '0.7rem', color: 'var(--text-muted)', whiteSpace: 'nowrap', textOverflow: 'ellipsis', overflow: 'hidden' }}>
              {user?.email}
            </div>
          </div>
          <button
            onClick={logout}
            className="btn btn-ghost"
            style={{ padding: '6px', color: 'var(--text-muted)' }}
            title="Sign out"
          >
            <LogOut size={16} />
          </button>
        </div>
      </aside>

      {/* 2. Main Content Area */}
      <main style={{ flex: 1, display: 'flex', flexDirection: 'column', overflowY: 'auto' }}>
        {/* Top Bar Header */}
        <header style={{
          height: '68px',
          borderBottom: '1px solid var(--border-subtle)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '0 32px',
          backgroundColor: 'rgba(8, 11, 18, 0.6)',
          backdropFilter: 'blur(12px)',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <h2 style={{ fontSize: '1.25rem', fontWeight: 600 }}>
              {activeTab === 'dashboard' && 'System Overview & Telemetry'}
              {activeTab === 'users' && 'User Directory & Access Control'}
              {activeTab === 'departments' && 'Organizational Units'}
              {activeTab === 'roles' && 'Role-Based Access Control (RBAC)'}
            </h2>
          </div>

          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <div style={{ display: 'flex', gap: '6px' }}>
              {roles.map((r) => (
                <span key={r} className="badge badge-role">
                  {r}
                </span>
              ))}
            </div>
            {isAdmin() && (
              <span className="badge badge-active" style={{ fontSize: '0.7rem' }}>
                SUPER ADMIN
              </span>
            )}
            <button
              onClick={fetchLiveCounts}
              className="btn btn-secondary"
              style={{ padding: '6px 12px', fontSize: '0.8rem' }}
              disabled={isLoadingData}
            >
              <RefreshCw size={14} className={isLoadingData ? 'spin' : ''} />
              <span>Refresh</span>
            </button>
          </div>
        </header>

        {/* Content Body */}
        <div style={{ padding: '32px', flex: 1 }}>
          {/* TAB 1: DASHBOARD */}
          {activeTab === 'dashboard' && (
            <DashboardView
              users={usersList}
              departments={deptsList}
              roles={rolesList}
              totalUserCount={userCount}
              isLoading={isLoadingData}
              onNavigateToUsers={() => setActiveTab('users')}
              onRefresh={fetchLiveCounts}
            />
          )}

          {/* TAB 2: USERS DIRECTORY (Full CRUD & Filtering) */}
          {activeTab === 'users' && (
            <UsersView />
          )}

          {/* TAB 3: DEPARTMENTS DIRECTORY (Full CRUD & Relational Integrity Guard) */}
          {activeTab === 'departments' && (
            <DepartmentsView
              departments={deptsList}
              users={usersList}
              onDataChanged={fetchLiveCounts}
            />
          )}

          {/* TAB 4: ROLES DIRECTORY (Full CRUD & RBAC Protection) */}
          {activeTab === 'roles' && (
            <RolesView />
          )}
        </div>
      </main>
    </div>
  );
};

export const App: React.FC = () => {
  return (
    <AuthProvider>
      <AppContent />
    </AuthProvider>
  );
};

const AppContent: React.FC = () => {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <div style={{ color: 'var(--text-muted)', fontSize: '0.9rem' }}>Restoring workspace session...</div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <LoginView />;
  }

  return <MainLayout />;
};

export default App;
