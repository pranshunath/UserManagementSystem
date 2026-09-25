import React, { useEffect, useState } from 'react';
import {
  userApi,
  departmentApi,
  roleApi,
} from '../api/client';

import type {
  User,
  Department,
  Role,
} from '../api/types';

const UsersView: React.FC = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [sortField, setSortField] = useState<
    'name' | 'department' | 'status' | 'created_at'
  >('name');

  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');


  const [currentPage, setCurrentPage] = useState(1);
  const usersPerPage = 10;

  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');


  const [showCreate, setShowCreate] = useState(false);

  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [departmentId, setDepartmentId] = useState('');
  const [selectedRoleIds, setSelectedRoleIds] = useState<number[]>([]);
  const [status, setStatus] = useState<'active' | 'inactive'>('active');

  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [editName, setEditName] = useState('');
  const [editEmail, setEditEmail] = useState('');
  const [editDepartmentId, setEditDepartmentId] = useState('');
  const [editRoleIds, setEditRoleIds] = useState<number[]>([]);
  const [editStatus, setEditStatus] =
    useState<'active' | 'inactive'>('active');

  const [message, setMessage] = useState('');
  const [error, setError] = useState('');

  const loadData = async () => {
    try {
      setLoading(true);
      setError('');

      const [userResult, departmentResult, roleResult] =
        await Promise.all([
          userApi.list({
            search: search || undefined,
            status: statusFilter || undefined,
          }),
          departmentApi.list(),
          roleApi.list(),
        ]);

      setUsers(userResult.users);
      setDepartments(departmentResult);
      setRoles(roleResult);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Failed to load users.'
      );
    } finally {
      setLoading(false);
    }
  };
  const handleSort = (
    field: 'name' | 'department' | 'status' | 'created_at'
  ) => {
    if (sortField === field) {
      setSortDirection(
        sortDirection === 'asc' ? 'desc' : 'asc'
      );
    } else {
      setSortField(field);
      setSortDirection('asc');
    }
  };

  useEffect(() => {
    setCurrentPage(1);
    loadData();
  }, [search, statusFilter]);

  const resetCreateForm = () => {
    setName('');
    setEmail('');
    setPassword('');
    setShowPassword(false);
    setDepartmentId('');
    setSelectedRoleIds([]);
    setStatus('active');
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (saving) return;
    if (!name.trim() || !email.trim() || !password) {
      setError('Name, email and password are required.');
      return;
    }

    if (password.length < 8) {
      setError('Password must be at least 8 characters.');
      return;
    }

    try {
      setSaving(true);
      setError('');
      setMessage('');

      await userApi.create({
        name: name.trim(),
        email: email.trim(),
        password,
        department_id: departmentId
          ? Number(departmentId)
          : null,
        role_ids: selectedRoleIds,
        status,
      });

      resetCreateForm();
      setShowCreate(false);

      setMessage('User created successfully.');
      await loadData();
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Failed to create user.'
      );
    } finally {
      setSaving(false);
    }
  };

  const startEdit = (user: User) => {
    setEditingUser(user);
    setEditName(user.name);
    setEditEmail(user.email);
    setEditDepartmentId(
      user.department_id ? String(user.department_id) : ''
    );
    setEditRoleIds(user.roles?.map((role) => role.id) || []);
    setEditStatus(user.status);

    setError('');
    setMessage('');
  };

  const cancelEdit = () => {
    setEditingUser(null);
  };

  const handleUpdate = async () => {
    if (!editingUser) return;

    try {
      setSaving(true);
      setError('');
      setMessage('');

      await userApi.update(editingUser.id, {
        name: editName.trim(),
        email: editEmail.trim(),
        department_id: editDepartmentId
          ? Number(editDepartmentId)
          : null,
        role_ids: editRoleIds,
        status: editStatus,
      });

      setEditingUser(null);
      setMessage('User updated successfully.');

      await loadData();
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Failed to update user.'
      );
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (user: User) => {
    const confirmed = window.confirm(
      `Delete user "${user.name}"?`
    );

    if (!confirmed) return;

    try {
      setSaving(true);
      setError('');
      setMessage('');

      await userApi.delete(user.id);

      setMessage('User deleted successfully.');
      await loadData();
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Failed to delete user.'
      );
    } finally {
      setSaving(false);
    }
  };

  const toggleRole = (
    roleId: number,
    currentIds: number[],
    setter: React.Dispatch<React.SetStateAction<number[]>>
  ) => {
    setter(
      currentIds.includes(roleId)
        ? currentIds.filter((id) => id !== roleId)
        : [...currentIds, roleId]
    );
  };
  const getDepartmentName = (user: User) => {
    if (user.department?.name) {
      return user.department.name;
    }

    const department = departments.find(
      (item) => item.id === user.department_id
    );

    return department?.name || 'No department';
  };
  const sortedUsers = [...users].sort((a, b) => {
    let valueA = '';
    let valueB = '';

    if (sortField === 'name') {
      valueA = a.name.toLowerCase();
      valueB = b.name.toLowerCase();
    } else if (sortField === 'department') {
      valueA = getDepartmentName(a).toLowerCase();
      valueB = getDepartmentName(b).toLowerCase();
    } else if (sortField === 'status') {
      valueA = a.status.toLowerCase();
      valueB = b.status.toLowerCase();
    } else if (sortField === 'created_at') {
      valueA = a.created_at;
      valueB = b.created_at;
    }

    const comparison = valueA.localeCompare(valueB);

    return sortDirection === 'asc'
      ? comparison
      : -comparison;
  });

  const totalPages = Math.ceil(
    sortedUsers.length / usersPerPage
  );

  const startIndex =
    (currentPage - 1) * usersPerPage;

  const paginatedUsers = sortedUsers.slice(
    startIndex,
    startIndex + usersPerPage
  );


  return (
    <div
      style={{
        padding: '42px 52px',
        maxWidth: '1250px',
        margin: '0 auto',
      }}
    >
      {/* Header */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-end',
          marginBottom: '28px',
        }}
      >
        <div>
          <div
            style={{
              fontSize: '13px',
              color: '#8d9ab5',
              marginBottom: '8px',
              letterSpacing: '0.08em',
            }}
          >
            ACCESS MANAGEMENT
          </div>

          <h1
            style={{
              margin: 0,
              fontSize: '36px',
            }}
          >
            Users Directory
          </h1>

          <p
            style={{
              margin: '8px 0 0',
              color: '#8995aa',
              fontSize: '15px',
            }}
          >
            Manage users, departments, roles and account status.
          </p>
        </div>

        <button
          onClick={() => {
            setShowCreate((value) => !value);
            setError('');
            setMessage('');
          }}
          style={primaryButton}
        >
          {showCreate ? 'Close' : '+ Register User'}
        </button>
      </div>

      {/* Messages */}
      {message && (
        <div style={successMessage}>
          {message}
        </div>
      )}

      {error && (
        <div style={errorMessage}>
          {error}
        </div>
      )}

      {/* Create User */}
      {showCreate && (
        <div style={panel}>
          <div style={{ marginBottom: '22px' }}>
            <h2 style={{ margin: 0, fontSize: '21px' }}>
              Register New User
            </h2>

            <p style={subText}>
              Create an account and assign access roles.
            </p>
          </div>

          <form onSubmit={handleCreate}>
            <div
              style={{
                display: 'grid',
                gridTemplateColumns:
                  'repeat(auto-fit, minmax(260px, 1fr))',
                gap: '16px',
              }}
            >
              <Field label="Full name">
                <input
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Enter full name"
                  style={inputStyle}
                />
              </Field>

              <Field label="Email address *">
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="name@company.com"
                  autoComplete="email"
                  required
                  style={inputStyle}
                />
              </Field>

              <Field label="Password *">
                <div
                  style={{
                    position: 'relative',
                  }}
                >
                  <input
                    type={showPassword ? 'text' : 'password'}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="Minimum 8 characters"
                    minLength={8}
                    style={{
                      ...inputStyle,
                      paddingRight: '70px',
                    }}
                  />

                  <button
                    type="button"
                    onClick={() => setShowPassword((value) => !value)}
                    style={{
                      position: 'absolute',
                      right: '10px',
                      top: '50%',
                      transform: 'translateY(-50%)',
                      border: 'none',
                      background: 'transparent',
                      color: '#8d9ab5',
                      cursor: 'pointer',
                      fontSize: '12px',
                      fontWeight: 600,
                    }}
                  >
                    {showPassword ? 'Hide' : 'Show'}
                  </button>
                </div>
              </Field>

              <Field label="Department">
                <select
                  value={departmentId}
                  onChange={(e) =>
                    setDepartmentId(e.target.value)
                  }
                  style={inputStyle}
                >
                  <option value="">
                    No department
                  </option>

                  {departments.map((department) => (
                    <option
                      key={department.id}
                      value={department.id}
                    >
                      {department.name}
                    </option>
                  ))}
                </select>
              </Field>

              <Field label="Status">
                <select
                  value={status}
                  onChange={(e) =>
                    setStatus(
                      e.target.value as
                      | 'active'
                      | 'inactive'
                    )
                  }
                  style={inputStyle}
                >
                  <option value="active">Active</option>
                  <option value="inactive">Inactive</option>
                </select>
              </Field>

              <Field label="Roles">
                <div
                  style={{
                    display: 'flex',
                    flexWrap: 'wrap',
                    gap: '8px',
                    paddingTop: '4px',
                  }}
                >
                  {roles.map((role) => {
                    const selected =
                      selectedRoleIds.includes(role.id);

                    return (
                      <button
                        key={role.id}
                        type="button"
                        onClick={() =>
                          toggleRole(
                            role.id,
                            selectedRoleIds,
                            setSelectedRoleIds
                          )
                        }
                        style={{
                          ...roleButton,
                          ...(selected
                            ? selectedRoleButton
                            : {}),
                        }}
                      >
                        {role.name}
                      </button>
                    );
                  })}
                </div>
              </Field>
            </div>

            <div
              style={{
                display: 'flex',
                justifyContent: 'flex-end',
                marginTop: '22px',
              }}
            >
              <button
                type="submit"
                disabled={saving}
                style={{
                  ...primaryButton,
                  opacity: saving ? 0.65 : 1,
                  cursor: saving ? 'not-allowed' : 'pointer',
                }}
              >
                {saving ? 'Creating...' : 'Create User'}
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Filters */}
      <div
        style={{
          display: 'flex',
          alignItems: 'flex-end',
          gap: '12px',
          marginBottom: '20px',
          flexWrap: 'wrap',
        }}
      >
        <div
          style={{
            position: 'relative',
            flex: 1,
            minWidth: '240px',
          }}
        >
          <span
            style={{
              position: 'absolute',
              left: '13px',
              top: '50%',
              transform: 'translateY(-50%)',
              color: '#6f7b91',
              fontSize: '16px',
              pointerEvents: 'none',
            }}
          >
            🔍
          </span>

          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search users by name or email..."
            style={{
              ...inputStyle,
              paddingLeft: '40px',
              paddingRight: search ? '40px' : '13px',
            }}
          />

          {search && (
            <button
              type="button"
              onClick={() => setSearch('')}
              style={{
                position: 'absolute',
                right: '10px',
                top: '50%',
                transform: 'translateY(-50%)',
                border: 'none',
                background: 'transparent',
                color: '#7f8ba3',
                cursor: 'pointer',
                fontSize: '18px',
                lineHeight: 1,
                padding: '4px',
              }}
              title="Clear search"
            >
              ×
            </button>
          )}
        </div>

        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: '5px',
          }}
        >
          <label
            style={{
              fontSize: '11px',
              color: '#707d94',
              fontWeight: 600,
              letterSpacing: '0.05em',
            }}
          >
            STATUS
          </label>

          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            style={{
              ...inputStyle,
              width: '180px',
            }}
          >
            <option value="">All statuses</option>
            <option value="active">Active</option>
            <option value="inactive">Inactive</option>
          </select>
        </div>

        <button
          onClick={loadData}
          style={secondaryButton}
        >
          ↻ Refresh
        </button>
        <button
          type="button"
          onClick={() => {
            setSearch('');
            setStatusFilter('');
            setSortField('name');
            setSortDirection('asc');
            setCurrentPage(1);
          }}
          style={secondaryButton}
        >
          Clear Filters
        </button>
      </div>

      {/* Directory */}
      <div style={panel}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            marginBottom: '20px',
          }}
        >
          <div>
            <h2
              style={{
                margin: 0,
                fontSize: '21px',
              }}
            >
              All Users
            </h2>

            <p style={subText}>
              {users.length} user{users.length !== 1 ? 's' : ''} found
              {' · '}
              <span style={{ color: '#32d6a0' }}>
                {users.filter((user) => user.status === 'active').length} active
              </span>
              {' · '}
              <span style={{ color: '#8995aa' }}>
                {users.filter((user) => user.status === 'inactive').length} inactive
              </span>
            </p>
          </div>
        </div>

        {loading ? (
          <div style={emptyState}>
            Loading users...
          </div>
        ) : users.length === 0 ? (
          <div style={emptyState}>
            <div
              style={{
                fontSize: '32px',
                marginBottom: '10px',
              }}
            >
              ◯
            </div>

            <strong>No users found</strong>

            <div
              style={{
                marginTop: '6px',
                color: '#7f8ba3',
              }}
            >
              Try changing your search or filters.
            </div>
          </div>
        ) : (
          <div
            style={{
              overflowX: 'auto',
            }}
          >
            <table
              style={{
                width: '100%',
                borderCollapse: 'collapse',
              }}
            >
              <thead
                style={{
                  position: 'sticky',
                  top: 0,
                  zIndex: 2,
                  background: '#111725',
                }}
              >
                <tr>
                  <th style={tableHeader}>
                    <button
                      type="button"
                      onClick={() => handleSort('name')}
                      style={sortButton}
                    >
                      USER
                      {sortField === 'name' &&
                        (sortDirection === 'asc' ? ' ↑' : ' ↓')}
                    </button>
                  </th>

                  <th style={tableHeader}>
                    <button
                      type="button"
                      onClick={() => handleSort('department')}
                      style={sortButton}
                    >
                      DEPARTMENT
                      {sortField === 'department' &&
                        (sortDirection === 'asc' ? ' ↑' : ' ↓')}
                    </button>
                  </th>

                  <th style={tableHeader}>
                    ROLES
                  </th>

                  <th style={tableHeader}>
                    <button
                      type="button"
                      onClick={() => handleSort('status')}
                      style={sortButton}
                    >
                      STATUS
                      {sortField === 'status' &&
                        (sortDirection === 'asc' ? ' ↑' : ' ↓')}
                    </button>
                  </th>

                  <th style={tableHeader}>
                    <button
                      type="button"
                      onClick={() => handleSort('created_at')}
                      style={sortButton}
                    >
                      DATE
                      {sortField === 'created_at' &&
                        (sortDirection === 'asc' ? ' ↑' : ' ↓')}
                    </button>
                  </th>

                  <th style={tableHeader}>
                    ACTIONS
                  </th>
                </tr>
              </thead>

              <tbody>
                {paginatedUsers.map((user) => (
                  <tr
                    key={user.id}
                    style={{
                      transition: 'background 0.15s ease',
                      background: user.id % 2 === 0 ? '#111725' : '#0f1420',
                    }}
                    onMouseEnter={(e) => {
                      e.currentTarget.style.background = '#171d2b';
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.background = 'transparent';
                    }}
                  >
                    <td style={tableCell}>
                      <div
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: '12px',
                        }}
                      >
                        <div style={avatar}>
                          {user.name
                            .charAt(0)
                            .toUpperCase()}
                        </div>

                        <div>
                          <div
                            style={{
                              fontWeight: 600,
                              color: '#edf1f7',
                            }}
                          >
                            {user.name}
                          </div>

                          <div
                            style={{
                              fontSize: '13px',
                              color: '#7f8ba3',
                              marginTop: '3px',
                            }}
                          >
                            {user.email}
                          </div>
                        </div>
                      </div>
                    </td>

                    <td style={tableCell}>
                      {getDepartmentName(user)}
                    </td>

                    <td style={tableCell}>
                      <div
                        style={{
                          display: 'flex',
                          flexWrap: 'wrap',
                          gap: '6px',
                        }}
                      >
                        {user.roles?.length ? (
                          user.roles.map((role) => (
                            <span
                              key={role.id}
                              style={roleBadge}
                            >
                              {role.name}
                            </span>
                          ))
                        ) : (
                          <span
                            style={{
                              color: '#6f7b91',
                            }}
                          >
                            No roles
                          </span>
                        )}
                      </div>
                    </td>

                    <td style={tableCell}>
                      <span
                        style={{
                          ...statusBadge,
                          ...(user.status === 'active'
                            ? activeBadge
                            : inactiveBadge),
                        }}
                      >
                        <span>●</span>
                        {user.status}
                      </span>
                    </td>

                    <td style={tableCell}>
                      {new Date(user.created_at).toLocaleDateString()}
                    </td>

                    <td style={tableCell}>
                      <div
                        style={{
                          display: 'flex',
                          gap: '8px',
                        }}
                      >
                        <button
                          onClick={() => startEdit(user)}
                          style={secondarySmallButton}
                        >
                          Edit
                        </button>

                        <button
                          onClick={() => handleDelete(user)}
                          disabled={saving}
                          style={deleteButton}
                        >
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {totalPages > 1 && (
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'center',
                  alignItems: 'center',
                  gap: '12px',
                  marginTop: '20px',
                }}
              >
                <button
                  onClick={() =>
                    setCurrentPage((page) => Math.max(page - 1, 1))
                  }
                  disabled={currentPage === 1}
                  style={secondaryButton}
                >
                  Previous
                </button>

                <span
                  style={{
                    color: '#9ba6bc',
                    fontSize: '14px',
                  }}
                >
                  Page {currentPage} of {totalPages}
                </span>

                <button
                  onClick={() =>
                    setCurrentPage((page) =>
                      Math.min(page + 1, totalPages)
                    )
                  }
                  disabled={currentPage === totalPages}
                  style={secondaryButton}
                >
                  Next
                </button>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Edit Modal */}
      {editingUser && (
        <div style={modalOverlay}>
          <div style={modal}>
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                marginBottom: '22px',
              }}
            >
              <div>
                <h2 style={{ margin: 0 }}>
                  Edit User
                </h2>

                <p style={subText}>
                  Update account details and access.
                </p>
              </div>

              <button
                onClick={cancelEdit}
                style={closeButton}
              >
                ×
              </button>
            </div>

            <Field label="Full name *">
              <input
                value={editName}
                onChange={(e) =>
                  setEditName(e.target.value)
                }
                style={inputStyle}
              />
            </Field>

            <Field label="Email address *">
              <input
                type="email"
                value={editEmail}
                onChange={(e) =>
                  setEditEmail(e.target.value)
                }
                style={inputStyle}
              />
            </Field>

            <Field label="Department">
              <select
                value={editDepartmentId}
                onChange={(e) =>
                  setEditDepartmentId(e.target.value)
                }
                style={inputStyle}
              >
                <option value="">No department</option>

                {departments.map((department) => (
                  <option
                    key={department.id}
                    value={department.id}
                  >
                    {department.name}
                  </option>
                ))}
              </select>
            </Field>

            <Field label="Status">
              <select
                value={editStatus}
                onChange={(e) =>
                  setEditStatus(
                    e.target.value as
                    | 'active'
                    | 'inactive'
                  )
                }
                style={inputStyle}
              >
                <option value="active">Active</option>
                <option value="inactive">Inactive</option>
              </select>
            </Field>

            <Field label="Roles">
              <div
                style={{
                  display: 'flex',
                  flexWrap: 'wrap',
                  gap: '8px',
                }}
              >
                {roles.map((role) => {
                  const selected =
                    editRoleIds.includes(role.id);

                  return (
                    <button
                      key={role.id}
                      type="button"
                      onClick={() =>
                        toggleRole(
                          role.id,
                          editRoleIds,
                          setEditRoleIds
                        )
                      }
                      style={{
                        ...roleButton,
                        ...(selected
                          ? selectedRoleButton
                          : {}),
                      }}
                    >
                      {role.name}
                    </button>
                  );
                })}
              </div>
            </Field>

            <div
              style={{
                display: 'flex',
                justifyContent: 'flex-end',
                gap: '10px',
                marginTop: '26px',
              }}
            >
              <button
                onClick={cancelEdit}
                style={secondaryButton}
              >
                Cancel
              </button>

              <button
                onClick={handleUpdate}
                disabled={saving}
                style={primaryButton}
              >
                {saving ? 'Saving...' : 'Save Changes'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

/* ---------------------------------------------------------
   Small reusable field component
--------------------------------------------------------- */

const Field = ({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) => (
  <div>
    <label
      style={{
        display: 'block',
        marginBottom: '7px',
        color: '#9ba6bc',
        fontSize: '13px',
      }}
    >
      {label}
    </label>

    {children}
  </div>
);

/* ---------------------------------------------------------
   Styles
--------------------------------------------------------- */

const inputStyle: React.CSSProperties = {
  width: '100%',
  boxSizing: 'border-box',
  padding: '11px 13px',
  borderRadius: '9px',
  border: '1px solid #30384d',
  background: '#0c111d',
  color: '#edf1f7',
  outline: 'none',
  fontSize: '14px',
};

const panel: React.CSSProperties = {
  background: '#111725',
  border: '1px solid #252d40',
  borderRadius: '16px',
  padding: '24px',
  marginBottom: '24px',
};

const primaryButton: React.CSSProperties = {
  padding: '10px 17px',
  borderRadius: '9px',
  border: 'none',
  background: '#5b4ee8',
  color: '#fff',
  fontWeight: 600,
  cursor: 'pointer',
};

const secondaryButton: React.CSSProperties = {
  padding: '10px 16px',
  borderRadius: '9px',
  border: '1px solid #353e55',
  background: '#171d2b',
  color: '#dce2ed',
  cursor: 'pointer',
  fontWeight: 500,
  transition: 'all 0.2s ease',
};

const secondarySmallButton: React.CSSProperties = {
  padding: '7px 12px',
  borderRadius: '7px',
  border: '1px solid #30384d',
  background: '#171d2b',
  color: '#dce2ed',
  cursor: 'pointer',
};

const deleteButton: React.CSSProperties = {
  padding: '7px 12px',
  borderRadius: '7px',
  border: '1px solid rgba(255, 80, 100, 0.25)',
  background: 'rgba(255, 80, 100, 0.08)',
  color: '#ff7180',
  cursor: 'pointer',
};

const roleButton: React.CSSProperties = {
  padding: '7px 11px',
  borderRadius: '20px',
  border: '1px solid #30384d',
  background: '#171d2b',
  color: '#9ba6bc',
  cursor: 'pointer',
  fontSize: '13px',
};

const selectedRoleButton: React.CSSProperties = {
  background: 'rgba(91, 78, 232, 0.18)',
  border: '1px solid rgba(91, 78, 232, 0.55)',
  color: '#aaa3ff',
};

const tableHeader: React.CSSProperties = {
  textAlign: 'left',
  padding: '13px 14px',
  borderBottom: '1px solid #30384d',
  color: '#9ba6bc',
  fontSize: '11px',
  fontWeight: 700,
  letterSpacing: '0.07em',
  background: '#111725',
};
const sortButton: React.CSSProperties = {
  border: 'none',
  background: 'transparent',
  color: 'inherit',
  font: 'inherit',
  padding: '4px 6px',
  margin: '-4px -6px',
  cursor: 'pointer',
  textAlign: 'left',
  fontWeight: 600,
  transition: 'color 0.2s ease',
};

const tableCell: React.CSSProperties = {
  padding: '16px 14px',
  borderBottom: '1px solid #1f2738',
  color: '#b9c2d2',
  fontSize: '14px',
};

const avatar: React.CSSProperties = {
  width: '38px',
  height: '38px',
  borderRadius: '50%',
  background: 'linear-gradient(135deg, #5b4ee8, #2d9fe8)',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  color: '#fff',
  fontWeight: 700,
};

const roleBadge: React.CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  padding: '5px 10px',
  borderRadius: '14px',
  background: 'rgba(91, 78, 232, 0.14)',
  border: '1px solid rgba(91, 78, 232, 0.28)',
  color: '#aaa3ff',
  fontSize: '12px',
  fontWeight: 600,
  lineHeight: 1,
};

const statusBadge: React.CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: '6px',
  padding: '5px 9px',
  borderRadius: '14px',
  fontSize: '12px',
  textTransform: 'capitalize',
};

const activeBadge: React.CSSProperties = {
  background: 'rgba(0, 200, 150, 0.12)',
  color: '#32d6a0',
  border: '1px solid rgba(0, 200, 150, 0.25)',
  fontWeight: 600,
};

const inactiveBadge: React.CSSProperties = {
  background: 'rgba(140, 150, 170, 0.10)',
  color: '#8995aa',
  border: '1px solid rgba(140, 150, 170, 0.20)',
  fontWeight: 600,
};

const successMessage: React.CSSProperties = {
  marginBottom: '20px',
  padding: '12px 16px',
  borderRadius: '10px',
  background: 'rgba(0, 200, 150, 0.10)',
  border: '1px solid rgba(0, 200, 150, 0.25)',
  color: '#32d6a0',
};

const errorMessage: React.CSSProperties = {
  marginBottom: '20px',
  padding: '12px 16px',
  borderRadius: '10px',
  background: 'rgba(255, 80, 100, 0.10)',
  border: '1px solid rgba(255, 80, 100, 0.25)',
  color: '#ff7180',
};

const subText: React.CSSProperties = {
  margin: '6px 0 0',
  color: '#7f8ba3',
  fontSize: '14px',
};

const emptyState: React.CSSProperties = {
  padding: '55px 20px',
  textAlign: 'center',
  color: '#8995aa',
};

const modalOverlay: React.CSSProperties = {
  position: 'fixed',
  inset: 0,
  background: 'rgba(0, 0, 0, 0.65)',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  padding: '20px',
  zIndex: 1000,
};

const modal: React.CSSProperties = {
  width: '100%',
  maxWidth: '600px',
  maxHeight: '90vh',
  overflowY: 'auto',
  background: '#111725',
  border: '1px solid #30384d',
  borderRadius: '18px',
  padding: '26px',
  boxSizing: 'border-box',
};

const closeButton: React.CSSProperties = {
  width: '34px',
  height: '34px',
  borderRadius: '8px',
  border: '1px solid #30384d',
  background: '#171d2b',
  color: '#aab4c6',
  fontSize: '22px',
  cursor: 'pointer',
};

export default UsersView;