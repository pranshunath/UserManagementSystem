import React, { useEffect, useState } from 'react';
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
  Sparkles,
  UserCheck,
  RefreshCw,
  CircleCheck,
  Gauge,
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
  onRefresh,
}) => {
  const [latency, setLatency] = useState<number | null>(null);
  const [isPinging, setIsPinging] = useState(false);

  const activeUsersCount = users.filter(
    (u) => u.status === 'active'
  ).length;

  const inactiveUsersCount = users.filter(
    (u) => u.status === 'inactive'
  ).length;

  const activeRate =
    users.length > 0
      ? Math.round((activeUsersCount / users.length) * 100)
      : 100;

  const deptDistribution = departments.map((department) => {
    const count = users.filter(
      (user) => user.department_id === department.id
    ).length;

    const percentage =
      users.length > 0
        ? Math.round((count / users.length) * 100)
        : 0;

    return {
      ...department,
      count,
      percentage,
    };
  });

  const unassignedCount = users.filter(
    (user) => !user.department_id
  ).length;

  const unassignedPercentage =
    users.length > 0
      ? Math.round((unassignedCount / users.length) * 100)
      : 0;

  const pingLatency = async () => {
    setIsPinging(true);

    const start = performance.now();

    try {
      await fetch(
        import.meta.env.VITE_HEALTH_URL ||
        'http://localhost:8080/health',
        {
          cache: 'no-store',
        }
      );

      const end = performance.now();

      setLatency(Math.round((end - start) * 10) / 10);
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
    <div className="dashboard-page fade-in">

      {/* HERO */}
      <section className="dashboard-hero">
        <div className="dashboard-hero-glow dashboard-hero-glow-one" />
        <div className="dashboard-hero-glow dashboard-hero-glow-two" />

        <div className="dashboard-hero-content">

          <div className="dashboard-hero-copy">

            <div className="dashboard-status-pill">
              <span className="dashboard-status-dot" />
              SYSTEM OPERATIONAL
            </div>

            <h1 className="dashboard-hero-title">
              Good overview.
              <br />
              <span>Everything at a glance.</span>
            </h1>

            <p className="dashboard-hero-description">
              Monitor users, organizational structure, access
              control and platform health from one place.
            </p>

          </div>

          <div className="dashboard-hero-actions">

            <div className="dashboard-latency-card">

              <div className="dashboard-latency-icon">
                <Activity size={19} />
              </div>

              <div className="dashboard-latency-content">
                <span>API LATENCY</span>

                <strong>
                  {latency !== null
                    ? `${latency} ms`
                    : 'Unavailable'}
                </strong>
              </div>

              <Gauge
                size={20}
                className="dashboard-latency-gauge"
              />

            </div>

            <div className="dashboard-hero-buttons">

              <button
                onClick={pingLatency}
                disabled={isPinging}
                className="dashboard-primary-button"
              >
                <Zap size={17} />

                {isPinging
                  ? 'Checking...'
                  : 'Check API'}
              </button>

              <button
                onClick={onRefresh}
                disabled={isLoading}
                className="dashboard-icon-button"
                title="Refresh dashboard"
              >
                <RefreshCw
                  size={18}
                  className={
                    isLoading
                      ? 'dashboard-spin'
                      : ''
                  }
                />
              </button>

            </div>

          </div>

        </div>
      </section>

      {/* METRICS */}
      <section className="dashboard-metrics">

        {/* Total Users */}
        <div className="dashboard-stat-card stat-indigo">

          <div className="dashboard-stat-top">

            <div>
              <span className="dashboard-stat-label">
                Total users
              </span>

              <div className="dashboard-stat-value">
                {totalUserCount}
              </div>
            </div>

            <div className="dashboard-stat-icon">
              <Users size={21} />
            </div>

          </div>

          <div className="dashboard-stat-footer">
            <UserCheck size={14} />

            <span>
              {activeUsersCount} active accounts
            </span>
          </div>

        </div>

        {/* Departments */}
        <div className="dashboard-stat-card stat-cyan">

          <div className="dashboard-stat-top">

            <div>
              <span className="dashboard-stat-label">
                Departments
              </span>

              <div className="dashboard-stat-value">
                {departments.length}
              </div>
            </div>

            <div className="dashboard-stat-icon">
              <Building2 size={21} />
            </div>

          </div>

          <div className="dashboard-stat-footer neutral">
            <CircleCheck size={14} />

            <span>
              Organizational units
            </span>
          </div>

        </div>

        {/* Roles */}
        <div className="dashboard-stat-card stat-amber">

          <div className="dashboard-stat-top">

            <div>
              <span className="dashboard-stat-label">
                Security roles
              </span>

              <div className="dashboard-stat-value">
                {roles.length}
              </div>
            </div>

            <div className="dashboard-stat-icon">
              <ShieldCheck size={21} />
            </div>

          </div>

          <div className="dashboard-stat-footer neutral">
            <CircleCheck size={14} />

            <span>
              Access control roles
            </span>
          </div>

        </div>

        {/* Active Rate */}
        <div className="dashboard-stat-card stat-emerald">

          <div className="dashboard-stat-top">

            <div>
              <span className="dashboard-stat-label">
                Active rate
              </span>

              <div className="dashboard-stat-value">
                {activeRate}%
              </div>
            </div>

            <div className="dashboard-stat-icon">
              <TrendingUp size={21} />
            </div>

          </div>

          <div className="dashboard-stat-footer success">
            <TrendingUp size={14} />

            <span>
              {inactiveUsersCount === 0
                ? 'All accounts active'
                : `${inactiveUsersCount} inactive accounts`}
            </span>
          </div>

        </div>

      </section>

      {/* ANALYTICS */}
      <section className="dashboard-analytics">

        {/* Department Distribution */}
        <div className="dashboard-panel department-panel">

          <div className="dashboard-panel-header">

            <div className="dashboard-panel-title-wrap">

              <div className="dashboard-panel-icon cyan">
                <Building2 size={18} />
              </div>

              <div>
                <h2>
                  Department distribution
                </h2>

                <p>
                  Where your users are organized
                </p>
              </div>

            </div>

            <span className="dashboard-count-pill">
              <Users size={13} />
              {users.length} users
            </span>

          </div>

          {departments.length === 0 ? (

            <div className="dashboard-empty-state">

              <Building2 size={30} />

              <strong>
                No departments yet
              </strong>

              <span>
                Create departments to see user
                distribution here.
              </span>

            </div>

          ) : (

            <div className="department-list">

              {deptDistribution.map(
                (department, index) => (

                  <div
                    key={department.id}
                    className="department-row"
                  >

                    <div className="department-row-header">

                      <div className="department-name-wrap">

                        <span className="department-index">
                          {String(index + 1).padStart(
                            2,
                            '0'
                          )}
                        </span>

                        <strong>
                          {department.name}
                        </strong>

                      </div>

                      <span className="department-count">
                        {department.count} users
                        <b>
                          {department.percentage}%
                        </b>
                      </span>

                    </div>

                    <div className="department-progress">

                      <div
                        className="department-progress-fill"
                        style={{
                          width: `${department.percentage}%`,
                        }}
                      />

                    </div>

                  </div>
                )
              )}

              {unassignedCount > 0 && (

                <div className="department-row unassigned">

                  <div className="department-row-header">

                    <div className="department-name-wrap">

                      <span className="department-index">
                        --
                      </span>

                      <strong>
                        Unassigned
                      </strong>

                    </div>

                    <span className="department-count">
                      {unassignedCount} users
                      <b>
                        {unassignedPercentage}%
                      </b>
                    </span>

                  </div>

                  <div className="department-progress">

                    <div
                      className="department-progress-fill"
                      style={{
                        width:
                          `${unassignedPercentage}%`,
                      }}
                    />

                  </div>

                </div>

              )}

            </div>

          )}

        </div>

        {/* Account Health */}
        <div className="dashboard-panel health-panel">

          <div className="dashboard-panel-header">

            <div className="dashboard-panel-title-wrap">

              <div className="dashboard-panel-icon green">
                <ShieldCheck size={18} />
              </div>

              <div>
                <h2>
                  Account health
                </h2>

                <p>
                  Current account status
                </p>
              </div>

            </div>

            <span className="health-status">
              <span />
              Healthy
            </span>

          </div>

          <div className="health-main">

            <div
              className="health-ring"
              style={{
                background: `conic-gradient(
                  var(--emerald) ${activeRate}%,
                  rgba(244,63,94,0.14)
                  ${activeRate}% 100%
                )`,
              }}
            >

              <div className="health-ring-inner">

                <strong>
                  {activeRate}%
                </strong>

                <span>
                  ACTIVE
                </span>

              </div>

            </div>

            <div className="health-legend">

              <div className="health-legend-item">

                <span className="legend-dot active" />

                <div>
                  <strong>
                    {activeUsersCount}
                  </strong>

                  <span>
                    Active
                  </span>
                </div>

              </div>

              <div className="health-legend-item">

                <span className="legend-dot inactive" />

                <div>
                  <strong>
                    {inactiveUsersCount}
                  </strong>

                  <span>
                    Inactive
                  </span>
                </div>

              </div>

            </div>

          </div>

          <div className="health-checks">

            {[
              'REST Gateway connected',
              'Request tracing enabled',
              'Password data protected',
              'Soft-delete filtering enabled',
            ].map((item) => (

              <div
                key={item}
                className="health-check"
              >

                <CheckCircle2 size={14} />

                <span>
                  {item}
                </span>

              </div>

            ))}

          </div>

        </div>

      </section>

      {/* INFRASTRUCTURE */}
      <section className="dashboard-panel infrastructure-panel">

        <div className="dashboard-panel-header">

          <div className="dashboard-panel-title-wrap">

            <div className="dashboard-panel-icon purple">
              <Layers size={18} />
            </div>

            <div>
              <h2>
                Platform infrastructure
              </h2>

              <p>
                Distributed services currently connected
              </p>
            </div>

          </div>

          <span className="infrastructure-status">
            <span className="dashboard-status-dot" />
            ALL SYSTEMS ONLINE
          </span>

        </div>

        <div className="infrastructure-grid">

          {[
            {
              name: 'API Gateway',
              icon: <Server size={17} />,
              color: 'indigo',
              host: 'localhost:8080',
              description:
                'Gin REST gateway with Request ID & CORS',
            },
            {
              name: 'User Microservice',
              icon: <Layers size={17} />,
              color: 'cyan',
              host: 'localhost:50051',
              description:
                'Protobuf v3 over HTTP/2',
            },
            {
              name: 'Redis 7 Cache',
              icon: <Database size={17} />,
              color: 'rose',
              host: 'localhost:6380',
              description:
                'Cache-aside GetByID queries',
            },
            {
              name: 'PostgreSQL 16',
              icon: <Database size={17} />,
              color: 'green',
              host: 'localhost:5432',
              description:
                'ACID relational storage',
            },
          ].map((service) => (

            <div
              key={service.name}
              className={`infrastructure-card infra-${service.color}`}
            >

              <div className="infrastructure-card-top">

                <div className="infrastructure-icon">
                  {service.icon}
                </div>

                <span className="online-dot">
                  ONLINE
                </span>

              </div>

              <strong>
                {service.name}
              </strong>

              <span className="infrastructure-host">
                {service.host}
              </span>

              <p>
                {service.description}
              </p>

            </div>

          ))}

        </div>

      </section>

      {/* RECENT USERS */}
      <section className="dashboard-panel recent-users-panel">

        <div className="dashboard-panel-header">

          <div className="dashboard-panel-title-wrap">

            <div className="dashboard-panel-icon indigo">
              <Users size={18} />
            </div>

            <div>
              <h2>
                Recent users
              </h2>

              <p>
                Latest accounts in the directory
              </p>
            </div>

          </div>

          <button
            onClick={onNavigateToUsers}
            className="dashboard-outline-button"
          >
            Open User Directory
            <ArrowUpRight size={15} />
          </button>

        </div>

        <div className="dashboard-table-wrapper">

          <table className="dashboard-table">

            <thead>
              <tr>
                <th>ID</th>
                <th>USER</th>
                <th>DEPARTMENT</th>
                <th>STATUS</th>
                <th>ROLES</th>
                <th>CREATED</th>
              </tr>
            </thead>

            <tbody>

              {users.slice(0, 8).map((user) => (

                <tr key={user.id}>

                  <td className="user-id">
                    #{user.id}
                  </td>

                  <td>

                    <div className="user-cell">

                      <div className="user-avatar">
                        {user.name
                          .trim()
                          .charAt(0)
                          .toUpperCase()}
                      </div>

                      <div className="user-details">

                        <strong>
                          {user.name}
                        </strong>

                        <span>
                          {user.email}
                        </span>

                      </div>

                    </div>

                  </td>

                  <td>

                    {user.department ? (

                      <span className="table-badge department">
                        {user.department.name}
                      </span>

                    ) : (

                      <span className="table-muted">
                        Unassigned
                      </span>

                    )}

                  </td>

                  <td>

                    <span
                      className={`table-badge ${user.status === 'active'
                        ? 'active'
                        : 'inactive'
                        }`}
                    >
                      <span />
                      {user.status}
                    </span>

                  </td>

                  <td>

                    <div className="role-list">

                      {user.roles &&
                        user.roles.length > 0 ? (

                        user.roles.map((role) => (

                          <span
                            key={role.id}
                            className="table-badge role"
                          >
                            {role.name}
                          </span>

                        ))

                      ) : (

                        <span className="table-muted">
                          None
                        </span>

                      )}

                    </div>

                  </td>

                  <td>

                    <span className="created-date">

                      <Clock size={12} />

                      {user.created_at
                        ? new Date(
                          user.created_at
                        ).toLocaleDateString()
                        : 'Recent'}

                    </span>

                  </td>

                </tr>

              ))}

              {users.length === 0 && (

                <tr>

                  <td
                    colSpan={6}
                    className="dashboard-table-empty"
                  >

                    <Sparkles size={22} />

                    <span>
                      No users available yet.
                    </span>

                  </td>

                </tr>

              )}

            </tbody>

          </table>

        </div>

      </section>

    </div>
  );
};