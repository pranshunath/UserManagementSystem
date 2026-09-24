import React, { useEffect, useState } from 'react';
import { roleApi } from '../api/client';
import type { Role } from '../api/types';

const RolesView: React.FC = () => {
  const [roles, setRoles] = useState<Role[]>([]);
  const [roleName, setRoleName] = useState('');
  const [description, setDescription] = useState('');

  const [editingId, setEditingId] = useState<number | null>(null);
  const [editingName, setEditingName] = useState('');
  const [editingDescription, setEditingDescription] = useState('');

  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');

  const loadRoles = async () => {
    try {
      setLoading(true);
      setError('');

      const data = await roleApi.list();
      setRoles(data);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Failed to load roles'
      );
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadRoles();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!roleName.trim()) {
      setError('Role name is required.');
      return;
    }

    try {
      setSaving(true);
      setError('');
      setMessage('');

      await roleApi.create(
        roleName.trim(),
        description.trim()
      );

      setRoleName('');
      setDescription('');

      setMessage('Role created successfully.');
      await loadRoles();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Failed to create role'
      );
    } finally {
      setSaving(false);
    }
  };

  const startEdit = (role: Role) => {
    setEditingId(role.id);
    setEditingName(role.name);
    setEditingDescription(role.description || '');
    setMessage('');
    setError('');
  };

  const cancelEdit = () => {
    setEditingId(null);
    setEditingName('');
    setEditingDescription('');
  };

  const handleUpdate = async (id: number) => {
    if (!editingName.trim()) {
      setError('Role name is required.');
      return;
    }

    try {
      setSaving(true);
      setError('');
      setMessage('');

      await roleApi.update(
        id,
        editingName.trim(),
        editingDescription.trim()
      );

      cancelEdit();

      setMessage('Role updated successfully.');
      await loadRoles();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Failed to update role'
      );
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (id: number) => {
    const confirmed = window.confirm(
      'Are you sure you want to delete this role?'
    );

    if (!confirmed) return;

    try {
      setSaving(true);
      setError('');
      setMessage('');

      await roleApi.delete(id);

      setMessage('Role deleted successfully.');
      await loadRoles();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Failed to delete role'
      );
    } finally {
      setSaving(false);
    }
  };

  return (
    <div
      style={{
        padding: '42px 52px',
        maxWidth: '1200px',
        margin: '0 auto',
      }}
    >
      {/* Header */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-end',
          marginBottom: '32px',
        }}
      >
        <div>
          <div
            style={{
              fontSize: '14px',
              color: '#8d9ab5',
              marginBottom: '8px',
            }}
          >
            ACCESS CONTROL
          </div>

          <h1
            style={{
              margin: 0,
              fontSize: '36px',
              fontWeight: 700,
            }}
          >
            Roles
          </h1>

          <p
            style={{
              marginTop: '8px',
              color: '#8d9ab5',
              fontSize: '15px',
            }}
          >
            Manage roles and control access across the platform.
          </p>
        </div>

        <button
          onClick={loadRoles}
          disabled={loading}
          style={{
            padding: '10px 18px',
            borderRadius: '10px',
            border: '1px solid #30384d',
            background: '#151b29',
            color: '#e8ecf5',
            cursor: loading ? 'not-allowed' : 'pointer',
          }}
        >
          ↻ Refresh
        </button>
      </div>

      {/* Messages */}
      {message && (
        <div
          style={{
            marginBottom: '20px',
            padding: '12px 16px',
            borderRadius: '10px',
            background: 'rgba(0, 200, 150, 0.10)',
            border: '1px solid rgba(0, 200, 150, 0.25)',
            color: '#32d6a0',
          }}
        >
          {message}
        </div>
      )}

      {error && (
        <div
          style={{
            marginBottom: '20px',
            padding: '12px 16px',
            borderRadius: '10px',
            background: 'rgba(255, 80, 100, 0.10)',
            border: '1px solid rgba(255, 80, 100, 0.25)',
            color: '#ff7180',
          }}
        >
          {error}
        </div>
      )}

      {/* Create Role */}
      <div
        style={{
          background: '#111725',
          border: '1px solid #252d40',
          borderRadius: '16px',
          padding: '24px',
          marginBottom: '32px',
        }}
      >
        <div style={{ marginBottom: '20px' }}>
          <h2
            style={{
              margin: 0,
              fontSize: '20px',
            }}
          >
            Create Role
          </h2>

          <p
            style={{
              margin: '6px 0 0',
              color: '#7f8ba3',
              fontSize: '14px',
            }}
          >
            Add a new access role to your organization.
          </p>
        </div>

        <form onSubmit={handleCreate}>
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: '1fr 2fr auto',
              gap: '14px',
              alignItems: 'end',
            }}
          >
            <div>
              <label
                style={{
                  display: 'block',
                  marginBottom: '7px',
                  fontSize: '13px',
                  color: '#9ba6bc',
                }}
              >
                Role name
              </label>

              <input
                value={roleName}
                onChange={(e) => setRoleName(e.target.value)}
                placeholder="e.g. Manager"
                style={inputStyle}
              />
            </div>

            <div>
              <label
                style={{
                  display: 'block',
                  marginBottom: '7px',
                  fontSize: '13px',
                  color: '#9ba6bc',
                }}
              >
                Description
              </label>

              <input
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Describe what this role can access"
                style={inputStyle}
              />
            </div>

            <button
              type="submit"
              disabled={saving}
              style={primaryButtonStyle}
            >
              {saving ? 'Saving...' : '+ Create Role'}
            </button>
          </div>
        </form>
      </div>

      {/* Roles Header */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: '16px',
        }}
      >
        <div>
          <h2 style={{ margin: 0, fontSize: '22px' }}>
            Existing Roles
          </h2>

          <p
            style={{
              margin: '5px 0 0',
              color: '#7f8ba3',
              fontSize: '14px',
            }}
          >
            {roles.length} role{roles.length !== 1 ? 's' : ''} configured
          </p>
        </div>
      </div>

      {/* Roles */}
      {loading ? (
        <div
          style={{
            padding: '40px',
            textAlign: 'center',
            color: '#8995aa',
          }}
        >
          Loading roles...
        </div>
      ) : roles.length === 0 ? (
        <div
          style={{
            padding: '50px',
            textAlign: 'center',
            background: '#111725',
            border: '1px solid #252d40',
            borderRadius: '16px',
            color: '#8995aa',
          }}
        >
          No roles found. Create your first role above.
        </div>
      ) : (
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))',
            gap: '16px',
          }}
        >
          {roles.map((role) => (
            <div
              key={role.id}
              style={{
                background: '#111725',
                border: '1px solid #252d40',
                borderRadius: '16px',
                padding: '20px',
              }}
            >
              {editingId === role.id ? (
                <>
                  <input
                    value={editingName}
                    onChange={(e) =>
                      setEditingName(e.target.value)
                    }
                    style={{
                      ...inputStyle,
                      marginBottom: '10px',
                    }}
                  />

                  <textarea
                    value={editingDescription}
                    onChange={(e) =>
                      setEditingDescription(e.target.value)
                    }
                    rows={3}
                    style={{
                      ...inputStyle,
                      resize: 'vertical',
                      fontFamily: 'inherit',
                      marginBottom: '14px',
                    }}
                  />

                  <div
                    style={{
                      display: 'flex',
                      gap: '8px',
                    }}
                  >
                    <button
                      onClick={() => handleUpdate(role.id)}
                      disabled={saving}
                      style={primaryButtonStyle}
                    >
                      Save
                    </button>

                    <button
                      onClick={cancelEdit}
                      style={secondaryButtonStyle}
                    >
                      Cancel
                    </button>
                  </div>
                </>
              ) : (
                <>
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'flex-start',
                    }}
                  >
                    <div>
                      <div
                        style={{
                          display: 'inline-block',
                          padding: '5px 10px',
                          borderRadius: '20px',
                          background: 'rgba(99, 88, 255, 0.12)',
                          color: '#9a91ff',
                          fontSize: '12px',
                          marginBottom: '12px',
                        }}
                      >
                        ROLE #{role.id}
                      </div>

                      <h3
                        style={{
                          margin: 0,
                          fontSize: '19px',
                        }}
                      >
                        {role.name}
                      </h3>
                    </div>
                  </div>

                  <p
                    style={{
                      color: '#8995aa',
                      fontSize: '14px',
                      lineHeight: 1.5,
                      minHeight: '42px',
                      margin: '12px 0 20px',
                    }}
                  >
                    {role.description || 'No description provided.'}
                  </p>

                  <div
                    style={{
                      display: 'flex',
                      gap: '8px',
                    }}
                  >
                    <button
                      onClick={() => startEdit(role)}
                      style={secondaryButtonStyle}
                    >
                      Edit
                    </button>

                    <button
                      onClick={() => handleDelete(role.id)}
                      disabled={saving}
                      style={deleteButtonStyle}
                    >
                      Delete
                    </button>
                  </div>
                </>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

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

const primaryButtonStyle: React.CSSProperties = {
  padding: '11px 18px',
  borderRadius: '9px',
  border: 'none',
  background: '#5b4ee8',
  color: 'white',
  fontWeight: 600,
  cursor: 'pointer',
  whiteSpace: 'nowrap',
};

const secondaryButtonStyle: React.CSSProperties = {
  padding: '9px 14px',
  borderRadius: '8px',
  border: '1px solid #30384d',
  background: '#171d2b',
  color: '#dce2ed',
  cursor: 'pointer',
};

const deleteButtonStyle: React.CSSProperties = {
  padding: '9px 14px',
  borderRadius: '8px',
  border: '1px solid rgba(255, 80, 100, 0.25)',
  background: 'rgba(255, 80, 100, 0.08)',
  color: '#ff7180',
  cursor: 'pointer',
};

export default RolesView;