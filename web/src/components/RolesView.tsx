import React, { useEffect, useState } from 'react';
import { roleApi } from '../api/client';
import type { Role } from '../api/types';

const RolesView: React.FC = () => {
  const [roles, setRoles] = useState<Role[]>([]);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [message, setMessage] = useState('');
  const [loading, setLoading] = useState(false);

  const loadRoles = async () => {
    try {
      const result = await roleApi.list();
      setRoles(result);
    } catch (error) {
      setMessage(
        error instanceof Error ? error.message : 'Failed to load roles'
      );
    }
  };

  useEffect(() => {
    loadRoles();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setMessage('');
    setLoading(true);

    try {
      await roleApi.create(
        name.trim(),
        description.trim()
      );

      setMessage('Role created successfully!');
      setName('');
      setDescription('');
      await loadRoles();
    } catch (error) {
      setMessage(
        error instanceof Error ? error.message : 'Failed to create role'
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ padding: '24px' }}>
      <h1>Roles</h1>

      <form
        onSubmit={handleCreate}
        style={{
          maxWidth: '500px',
          display: 'flex',
          flexDirection: 'column',
          gap: '12px',
          marginBottom: '24px',
        }}
      >
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Role name"
          required
          style={{ padding: '10px' }}
        />

        <input
          type="text"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Description"
          style={{ padding: '10px' }}
        />

        <button type="submit" disabled={loading}>
          {loading ? 'Creating...' : 'Create Role'}
        </button>
      </form>

      {message && <p>{message}</p>}

      <h2>Existing Roles</h2>

      {roles.map((role) => (
        <div key={role.id} style={{ marginBottom: '8px' }}>
          <strong>{role.name}</strong>
          {role.description ? ` — ${role.description}` : ''}
        </div>
      ))}
    </div>
  );
};

export default RolesView;