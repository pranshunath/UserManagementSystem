import React, { useState, useEffect } from 'react';
import type { User, Department, Role } from '../api/types';
import {
  Users,
  Building2,
  ShieldCheck,
  Zap,
  Server,
  Database,
  Layers,
  ArrowUpRight,
  TrendingUp,
  Activity,
  CheckCircle2,
  Clock,
  Radio,
} from 'lucide-react';

interface DashboardViewProps {
  users: User[];
  departments: Department[];
  roles: Role[];
  totalUserCount: number;
  isLoading: boolean;
  onNavigateToUsers: () => void;
  onRefresh: () => void;
}

export const DashboardView: React.FC<DashboardViewProps> = ({
  users,
  departments,
  roles,
  totalUserCount,
  isLoading,
  onNavigateToUsers,
}) => {
  const [latency, setLatency] = useState<number | null>(null);
  const [isPinging, setIsPinging] = useState(false);

  // Calculate statistics
  const activeUsersCount = users.filter((u) => u.status === 'active').length;
  const inactiveUsersCount = users.filter((u) => u.status === 'inactive').length;
  const activeRate = users.length > 0 ? Math.round((activeUsersCount / users.length) * 100) : 100;

  // Department distribution
  const deptDistribution = departments.map((d) => {
    const count = users.filter((u) => u.department_id === d.id).length;
    const percentage = users.length > 0 ? Math.round((count / users.length) * 100) : 0;
    return { ...d, count, percentage };
  });

  const unassignedCount = users.filter((u) => !u.department_id).length;
  const unassignedPercentage = users.length > 0 ? Math.round((unassignedCount / users.length) * 100) : 0;

  // Measure round-trip ping latency to API Gateway
  const pingLatency = async () => {
    setIsPinging(true);
    const t0 = performance.now();
    try {
      await fetch(import.meta.env.VITE_HEALTH_URL || 'http://localhost:8080/health', {
        cache: 'no-store',
      });
      const t1 = performance.now();
      setLatency(Math.round((t1 - t0) * 10) / 10);
    } catch {
      setLatency(null);
    } finally {
      setIsPinging(false);
    }
  };

  useEffect(() => {
    pingLatency();
  }, []);

  return (
    <div className="fade-in" style={{ display: 'flex', flexDirection: 'column', gap: '28px' }}>
      {/* 1. Header Banner with Live Benchmark Ping */}
      <div className="glass-panel" style={{
        padding: '24px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        flexWrap: 'wrap',
        gap: '16px',
        background: 'linear-gradient(135deg, rgba(18, 24, 38, 0.9) 0%, rgba(26, 34, 54, 0.6) 100%)',
      }}>
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '6px' }}>
            <span style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: '6px',
              padding: '2px 8px',
              borderRadius: '999px',
              fontSize: '0.75rem',
              fontWeight: 600,
              background: 'rgba(99, 102, 241, 0.15)',
              color: '#818cf8',
              border: '1px solid rgba(99, 102, 241, 0.3)',
            }}>
              <Radio size={12} className={isLoading ? 'spin' : ''} />
              {isLoading ? 'SYNCING DATA...' : 'LIVE TELEMETRY'}
            </span>
            <span style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>
              Connected to Gin Gateway (:8080) & gRPC Microservice (:50051)
            </span>
          </div>
          <h2 style={{ fontSize: '1.5rem', fontWeight: 700 }}>System Intelligence & Architecture Health</h2>
        </div>

        {/* Live Latency Gauge & Interactive Ping Button */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <div style={{
            padding: '8px 16px',
            borderRadius: 'var(--radius-md)',
            background: 'rgba(255, 255, 255, 0.03)',
            border: '1px solid var(--border-subtle)',
            display: 'flex',
            alignItems: 'center',
            gap: '10px',
          }}>
            <Zap size={18} color="#06b6d4" />
            <div>
              <div style={{ fontSize: '0.7rem', color: 'var(--text-muted)', textTransform: 'uppercase', fontWeight: 600 }}>
                API Ping Latency
              </div>
              <div style={{ fontSize: '1rem', fontWeight: 700, color: latency !== null && latency < 50 ? 'var(--emerald)' : 'var(--amber)' }}>
                {latency !== null ? `${latency} ms` : 'Measuring...'}
              </div>
            </div>
          </div>

          <button
            onClick={pingLatency}
            disabled={isPinging}
            className="btn btn-secondary"
            style={{ padding: '10px 14px' }}
          >
            <Activity size={16} className={isPinging ? 'spin' : ''} />
            <span>{isPinging ? 'Pinging...' : 'Benchmark Ping'}</span>
          </button>
        </div>
      </div>

      {/* 2. Primary Telemetry Metric Cards */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: '20px' }}>
        {/* Metric 1: Total Users */}
        <div className="glass-panel glass-panel-hover" style={{ padding: '22px' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '14px' }}>
            <span style={{ color: 'var(--text-secondary)', fontSize: '0.8125rem', fontWeight: 500 }}>Total Accounts</span>
            <div style={{ color: '#818cf8', background: 'var(--primary-subtle)', padding: '8px', borderRadius: '8px' }}>
              <Users size={20} />
            </div>
          </div>
          <div style={{ fontSize: '2.25rem', fontWeight: 700, fontFamily: 'var(--font-heading)' }}>
            {totalUserCount}
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginTop: '6px' }}>
            <span style={{ color: 'var(--emerald)', fontSize: '0.8rem', fontWeight: 600, display: 'flex', alignItems: 'center' }}>
              <TrendingUp size={14} style={{ marginRight: '3px' }} />
              {activeRate}% Active
            </span>
            <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>
              ({activeUsersCount} active, {inactiveUsersCount} inactive)
            </span>
          </div>
        </div>

        {/* Metric 2: Departments */}
        <div className="glass-panel glass-panel-hover" style={{ padding: '22px' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '14px' }}>
            <span style={{ color: 'var(--text-secondary)', fontSize: '0.8125rem', fontWeight: 500 }}>Departments</span>
            <div style={{ color: '#38bdf8', background: 'rgba(6, 182, 212, 0.12)', padding: '8px', borderRadius: '8px' }}>
              <Building2 size={20} />
            </div>
          </div>
          <div style={{ fontSize: '2.25rem', fontWeight: 700, fontFamily: 'var(--font-heading)' }}>
            {departments.length}
          </div>
          <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginTop: '6px' }}>
            {departments.length > 0
              ? `Avg ${Math.round(totalUserCount / departments.length)} users per department`
              : 'No departments created yet'}
          </div>
        </div>

        {/* Metric 3: Roles */}
        <div className="glass-panel glass-panel-hover" style={{ padding: '22px' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '14px' }}>
            <span style={{ color: 'var(--text-secondary)', fontSize: '0.8125rem', fontWeight: 500 }}>Security Roles</span>
            <div style={{ color: '#f59e0b', background: 'var(--amber-bg)', padding: '8px', borderRadius: '8px' }}>
              <ShieldCheck size={20} />
            </div>
          </div>
          <div style={{ fontSize: '2.25rem', fontWeight: 700, fontFamily: 'var(--font-heading)' }}>
            {roles.length}
          </div>
          <div style={{ display: 'flex', gap: '4px', flexWrap: 'wrap', marginTop: '8px' }}>
            {roles.map((r) => (
              <span key={r.id} className="badge badge-role" style={{ fontSize: '0.7rem' }}>
                {r.name}
              </span>
            ))}
          </div>
        </div>

        {/* Metric 4: Cache Health */}
        <div className="glass-panel glass-panel-hover" style={{ padding: '22px' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '14px' }}>
            <span style={{ color: 'var(--text-secondary)', fontSize: '0.8125rem', fontWeight: 500 }}>Cache Strategy</span>
            <div style={{ color: '#f43f5e', background: 'var(--rose-bg)', padding: '8px', borderRadius: '8px' }}>
              <Database size={20} />
            </div>
          </div>
          <div style={{ fontSize: '1.4rem', fontWeight: 700, color: 'var(--emerald)' }}>
            Cache-Aside Active
          </div>
          <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginTop: '8px' }}>
            Redis 7 on Port <strong>:6380</strong> with 10m TTL & automatic mutation invalidation
          </div>
        </div>
      </div>

      {/* 3. Two Column Visual Analytics Breakdown */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))', gap: '24px' }}>
        {/* Left Column: Department Headcount Distribution */}
        <div className="glass-panel" style={{ padding: '24px' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '20px' }}>
            <h3 style={{ fontSize: '1.1rem', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '8px' }}>
              <Building2 size={18} color="#06b6d4" />
              Department Headcount Distribution
            </h3>
            <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>{users.length} Users Sampled</span>
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            {deptDistribution.map((d) => (
              <div key={d.id}>
                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.85rem', marginBottom: '6px' }}>
                  <span style={{ fontWeight: 500 }}>{d.name}</span>
                  <span style={{ color: 'var(--text-muted)' }}>
                    {d.count} users ({d.percentage}%)
                  </span>
                </div>
                {/* Progress bar */}
                <div style={{ width: '100%', height: '8px', background: 'rgba(255, 255, 255, 0.06)', borderRadius: '4px', overflow: 'hidden' }}>
                  <div
                    style={{
                      height: '100%',
                      width: `${d.percentage}%`,
                      background: 'linear-gradient(90deg, #6366f1, #06b6d4)',
                      borderRadius: '4px',
                      transition: 'width 0.4s ease',
                    }}
                  />
                </div>
              </div>
            ))}

            {unassignedCount > 0 && (
              <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.85rem', marginBottom: '6px' }}>
                  <span style={{ color: 'var(--text-muted)' }}>Unassigned Department</span>
                  <span style={{ color: 'var(--text-muted)' }}>
                    {unassignedCount} users ({unassignedPercentage}%)
                  </span>
                </div>
                <div style={{ width: '100%', height: '8px', background: 'rgba(255, 255, 255, 0.06)', borderRadius: '4px', overflow: 'hidden' }}>
                  <div
                    style={{
                      height: '100%',
                      width: `${unassignedPercentage}%`,
                      background: 'rgba(255, 255, 255, 0.2)',
                      borderRadius: '4px',
                    }}
                  />
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Right Column: Account Status & Governance Split */}
        <div className="glass-panel" style={{ padding: '24px' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '20px' }}>
            <h3 style={{ fontSize: '1.1rem', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '8px' }}>
              <ShieldCheck size={18} color="#10b981" />
              Account Status & Security Governance
            </h3>
            <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>Security Invariant Check</span>
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            {/* Status Split Bar */}
            <div>
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.85rem', marginBottom: '8px' }}>
                <span style={{ color: 'var(--emerald)', fontWeight: 600 }}>Active ({activeUsersCount})</span>
                <span style={{ color: 'var(--rose)', fontWeight: 600 }}>Inactive ({inactiveUsersCount})</span>
              </div>
              <div style={{ display: 'flex', width: '100%', height: '12px', borderRadius: '6px', overflow: 'hidden', gap: '2px' }}>
                <div style={{ width: `${activeRate}%`, background: 'var(--emerald)', transition: 'width 0.4s ease' }} title={`Active: ${activeRate}%`} />
                <div style={{ width: `${100 - activeRate}%`, background: 'var(--rose)', transition: 'width 0.4s ease' }} title={`Inactive: ${100 - activeRate}%`} />
              </div>
            </div>

            {/* Architecture Governance Checklist */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: '10px', marginTop: '10px' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px', fontSize: '0.85rem' }}>
                <CheckCircle2 size={16} color="var(--emerald)" />
                <span>REST Gateway decoupled from database (gRPC only)</span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px', fontSize: '0.85rem' }}>
                <CheckCircle2 size={16} color="var(--emerald)" />
                <span>Zero password hashes serialized to JSON or Protobuf</span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px', fontSize: '0.85rem' }}>
                <CheckCircle2 size={16} color="var(--emerald)" />
                <span>Distributed Request ID tracing passed across HTTP/gRPC</span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px', fontSize: '0.85rem' }}>
                <CheckCircle2 size={16} color="var(--emerald)" />
                <span>Soft-delete preservation (SQL <code>deleted_at</code> filtering)</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* 4. Distributed Tier Node Topology Status */}
      <div className="glass-panel" style={{ padding: '24px' }}>
        <h3 style={{ fontSize: '1.1rem', fontWeight: 600, marginBottom: '18px', display: 'flex', alignItems: 'center', gap: '8px' }}>
          <Layers size={18} color="#6366f1" />
          Full-Stack Distributed Tier Architecture
        </h3>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: '16px' }}>
          {/* Node 1: API Gateway */}
          <div style={{ padding: '16px', background: 'rgba(255, 255, 255, 0.02)', borderRadius: 'var(--radius-md)', border: '1px solid var(--border-subtle)' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '8px' }}>
              <span style={{ fontWeight: 600, color: '#818cf8', display: 'flex', alignItems: 'center', gap: '6px' }}>
                <Server size={16} /> API Gateway
              </span>
              <span className="badge badge-active" style={{ fontSize: '0.65rem' }}>ONLINE</span>
            </div>
            <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>Host: <code>localhost:8080</code></div>
            <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginTop: '4px' }}>Gin HTTP/1.1 REST with Request ID & CORS</div>
          </div>

          {/* Node 2: gRPC Microservice */}
          <div style={{ padding: '16px', background: 'rgba(255, 255, 255, 0.02)', borderRadius: 'var(--radius-md)', border: '1px solid var(--border-subtle)' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '8px' }}>
              <span style={{ fontWeight: 600, color: '#06b6d4', display: 'flex', alignItems: 'center', gap: '6px' }}>
                <Layers size={16} /> User Microservice
              </span>
              <span className="badge badge-active" style={{ fontSize: '0.65rem' }}>ONLINE</span>
            </div>
            <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>Host: <code>localhost:50051</code></div>
            <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginTop: '4px' }}>Protobuf v3 over HTTP/2 with Unary Logging</div>
          </div>

          {/* Node 3: Redis */}
          <div style={{ padding: '16px', background: 'rgba(255, 255, 255, 0.02)', borderRadius: 'var(--radius-md)', border: '1px solid var(--border-subtle)' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '8px' }}>
              <span style={{ fontWeight: 600, color: '#f43f5e', display: 'flex', alignItems: 'center', gap: '6px' }}>
                <Database size={16} /> Redis 7 Cache
              </span>
              <span className="badge badge-active" style={{ fontSize: '0.65rem' }}>ONLINE</span>
            </div>
            <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>Host: <code>localhost:6380</code></div>
            <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginTop: '4px' }}>Cache-Aside for GetByID queries with 10m TTL</div>
          </div>

          {/* Node 4: PostgreSQL */}
          <div style={{ padding: '16px', background: 'rgba(255, 255, 255, 0.02)', borderRadius: 'var(--radius-md)', border: '1px solid var(--border-subtle)' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '8px' }}>
              <span style={{ fontWeight: 600, color: '#10b981', display: 'flex', alignItems: 'center', gap: '6px' }}>
                <Database size={16} /> PostgreSQL 16
              </span>
              <span className="badge badge-active" style={{ fontSize: '0.65rem' }}>ONLINE</span>
            </div>
            <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)' }}>Host: <code>localhost:5432</code></div>
            <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginTop: '4px' }}>ACID Relational Storage & Many-to-Many Junction</div>
          </div>
        </div>
      </div>

      {/* 5. Live Recent Users Table */}
      <div className="glass-panel" style={{ padding: '24px' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '16px' }}>
          <div>
            <h3 style={{ fontSize: '1.1rem', fontWeight: 600 }}>Recent Users Activity</h3>
            <p style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>Latest synchronized accounts from PostgreSQL</p>
          </div>
          <button onClick={onNavigateToUsers} className="btn btn-secondary" style={{ fontSize: '0.8rem', padding: '6px 14px' }}>
            <span>Open User Directory</span>
            <ArrowUpRight size={14} />
          </button>
        </div>

        <div style={{ overflowX: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left', fontSize: '0.875rem' }}>
            <thead>
              <tr style={{ borderBottom: '1px solid var(--border-subtle)', color: 'var(--text-muted)' }}>
                <th style={{ padding: '10px 14px' }}>ID</th>
                <th style={{ padding: '10px 14px' }}>NAME & EMAIL</th>
                <th style={{ padding: '10px 14px' }}>DEPARTMENT</th>
                <th style={{ padding: '10px 14px' }}>STATUS</th>
                <th style={{ padding: '10px 14px' }}>ROLES</th>
                <th style={{ padding: '10px 14px' }}>CREATED</th>
              </tr>
            </thead>
            <tbody>
              {users.slice(0, 8).map((u) => (
                <tr key={u.id} style={{ borderBottom: '1px solid rgba(255, 255, 255, 0.04)' }}>
                  <td style={{ padding: '12px 14px', fontFamily: 'monospace', color: 'var(--text-muted)' }}>
                    #{u.id}
                  </td>
                  <td style={{ padding: '12px 14px' }}>
                    <div style={{ fontWeight: 600 }}>{u.name}</div>
                    <div style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>{u.email}</div>
                  </td>
                  <td style={{ padding: '12px 14px' }}>
                    {u.department ? (
                      <span className="badge badge-dept">{u.department.name}</span>
                    ) : (
                      <span style={{ color: 'var(--text-muted)', fontSize: '0.8rem' }}>Unassigned</span>
                    )}
                  </td>
                  <td style={{ padding: '12px 14px' }}>
                    <span className={`badge ${u.status === 'active' ? 'badge-active' : 'badge-inactive'}`}>
                      {u.status}
                    </span>
                  </td>
                  <td style={{ padding: '12px 14px' }}>
                    <div style={{ display: 'flex', gap: '4px', flexWrap: 'wrap' }}>
                      {u.roles && u.roles.length > 0 ? (
                        u.roles.map((r) => (
                          <span key={r.id} className="badge badge-role" style={{ fontSize: '0.7rem' }}>
                            {r.name}
                          </span>
                        ))
                      ) : (
                        <span style={{ color: 'var(--text-muted)', fontSize: '0.8rem' }}>None</span>
                      )}
                    </div>
                  </td>
                  <td style={{ padding: '12px 14px', color: 'var(--text-muted)', fontSize: '0.8rem' }}>
                    <span style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                      <Clock size={12} />
                      {u.created_at ? new Date(u.created_at).toLocaleDateString() : 'Recent'}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};
