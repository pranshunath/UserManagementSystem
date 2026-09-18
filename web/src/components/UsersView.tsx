import React, { useState } from 'react';
import { userApi } from '../api/client';

const UsersView: React.FC = () => {
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [departmentId, setDepartmentId] = useState('');
  const [roleIds, setRoleIds] = useState('');
  const [status, setStatus] = useState<'active' | 'inactive'>('active');

  const [message, setMessage] = useState('');
  const [loading, setLoading] = useState(false);

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();

    setMessage('');
    setLoading(true);

    try {
      const payload = {
        name: name.trim(),
        email: email.trim(),
        password,
        department_id: departmentId
          ? Number(departmentId)
          : null,
        role_ids: roleIds
          ? roleIds.split(',').map((id) => Number(id.trim()))
          : [],
        status,
      };

      await userApi.create(payload);

      setMessage('User registered successfully!');

      // Clear form
      setName('');
      setEmail('');
      setPassword('');
      setDepartmentId('');
      setRoleIds('');
      setStatus('active');

    } catch (error) {
      setMessage(
        error instanceof Error
          ? error.message
          : 'Something went wrong'
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ padding: '24px' }}>
      <h1>Register New User</h1>

      <form
        onSubmit={handleRegister}
        style={{
          maxWidth: '500px',
          display: 'flex',
          flexDirection: 'column',
          gap: '12px',
        }}
      >
        {/* Name */}
        <div>
          <label>Name</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Enter user name"
            required
            style={{ width: '100%', padding: '10px' }}
          />
        </div>

        {/* Email */}
        <div>
          <label>Email</label>
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="Enter email"
            required
            style={{ width: '100%', padding: '10px' }}
          />
        </div>

        {/* Password */}
        <div>
          <label>Password</label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Enter password"
            required
            minLength={8}
            style={{ width: '100%', padding: '10px' }}
          />
        </div>

        {/* Department */}
        <div>
          <label>Department ID</label>
          <input
            type="number"
            value={departmentId}
            onChange={(e) => setDepartmentId(e.target.value)}
            placeholder="Example: 1"
            style={{ width: '100%', padding: '10px' }}
          />
        </div>

        {/* Roles */}
        <div>
          <label>Role IDs</label>
          <input
            type="text"
            value={roleIds}
            onChange={(e) => setRoleIds(e.target.value)}
            placeholder="Example: 1,2"
            style={{ width: '100%', padding: '10px' }}
          />
          <small>
            Enter role IDs separated by commas.
          </small>
        </div>

        {/* Status */}
        <div>
          <label>Status</label>

          <select
            value={status}
            onChange={(e) =>
              setStatus(e.target.value as 'active' | 'inactive')
            }
            style={{ width: '100%', padding: '10px' }}
          >
            <option value="active">Active</option>
            <option value="inactive">Inactive</option>
          </select>
        </div>

        {/* Submit */}
        <button
          type="submit"
          disabled={loading}
          style={{
            padding: '12px',
            cursor: loading ? 'not-allowed' : 'pointer',
          }}
        >
          {loading ? 'Registering...' : 'Register User'}
        </button>

        {/* Message */}
        {message && (
          <p>{message}</p>
        )}
      </form>
    </div>
  );
};

export default UsersView;