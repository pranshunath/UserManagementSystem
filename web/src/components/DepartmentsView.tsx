import React, { useState } from 'react';
import type { Department, User } from '../api/types';
import { departmentApi, ApiClientError } from '../api/client';
import { useAuth } from '../context/AuthContext';
import {
  Building2,
  Plus,
  Edit2,
  Trash2,
  Search,
  RotateCcw,
  AlertCircle,
  Users,
  X,
  Shield,
  Layers,
} from 'lucide-react';

interface DepartmentsViewProps {
  departments: Department[];
  users: User[];
  onDataChanged?: () => void;
}

export const DepartmentsView: React.FC<DepartmentsViewProps> = ({
  departments,
  users,
  onDataChanged,
}) => {
  const { isAdmin } = useAuth();
  const canManage = isAdmin();

  // Search filter
  const [search, setSearch] = useState<string>('');

  // Modal states
  const [isFormModalOpen, setIsFormModalOpen] =
    useState<boolean>(false);

  const [editingDept, setEditingDept] =
    useState<Department | null>(null);

  const [deleteConfirmDept, setDeleteConfirmDept] =
    useState<Department | null>(null);

  // Form inputs
  const [formName, setFormName] = useState<string>('');
  const [formDescription, setFormDescription] =
    useState<string>('');

  const [formError, setFormError] =
    useState<string | null>(null);

  const [formRequestId, setFormRequestId] =
    useState<string | null>(null);

  const [isSubmitting, setIsSubmitting] =
    useState<boolean>(false);

  // ---------------------------------------------------------
  // Filter departments
  // ---------------------------------------------------------

  const filteredDepartments = departments.filter((d) => {
    const q = search.toLowerCase().trim();

    if (!q) return true;

    return (
      d.name.toLowerCase().includes(q) ||
      (d.description &&
        d.description.toLowerCase().includes(q))
    );
  });

  // ---------------------------------------------------------
  // Calculate department headcount
  // ---------------------------------------------------------

  const getDeptHeadcount = (deptId: number): number => {
    return users.filter(
      (u) => u.department_id === deptId
    ).length;
  };

  // ---------------------------------------------------------
  // Get users for department
  // ---------------------------------------------------------

  const getDeptUsers = (deptId: number): User[] => {
    return users.filter(
      (u) => u.department_id === deptId
    );
  };

  // ---------------------------------------------------------
  // Open Create Modal
  // ---------------------------------------------------------

  const openCreateModal = () => {
    setEditingDept(null);
    setFormName('');
    setFormDescription('');
    setFormError(null);
    setFormRequestId(null);
    setIsFormModalOpen(true);
  };

  // ---------------------------------------------------------
  // Open Edit Modal
  // ---------------------------------------------------------

  const openEditModal = (department: Department) => {
    setEditingDept(department);
    setFormName(department.name);
    setFormDescription(
      department.description || ''
    );
    setFormError(null);
    setFormRequestId(null);
    setIsFormModalOpen(true);
  };

  // ---------------------------------------------------------
  // Close Form Modal
  // ---------------------------------------------------------

  const closeFormModal = () => {
    if (isSubmitting) return;

    setIsFormModalOpen(false);
    setEditingDept(null);
    setFormName('');
    setFormDescription('');
    setFormError(null);
    setFormRequestId(null);
  };

  // ---------------------------------------------------------
  // Submit Create / Edit
  // ---------------------------------------------------------

  const handleFormSubmit = async (
    e: React.FormEvent
  ) => {
    e.preventDefault();

    setFormError(null);
    setFormRequestId(null);

    if (!formName.trim()) {
      setFormError(
        'Department name is required.'
      );
      return;
    }

    setIsSubmitting(true);

    try {
      if (editingDept) {
        await departmentApi.update(
          editingDept.id,
          formName.trim(),
          formDescription.trim()
        );
      } else {
        await departmentApi.create(
          formName.trim(),
          formDescription.trim()
        );
      }

      setIsFormModalOpen(false);
      setEditingDept(null);
      setFormName('');
      setFormDescription('');

      onDataChanged?.();
    } catch (err: unknown) {
      if (err instanceof ApiClientError) {
        setFormError(err.message);

        if (err.requestId) {
          setFormRequestId(err.requestId);
        }
      } else {
        setFormError(
          'Operation failed. Please verify your input.'
        );
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  // ---------------------------------------------------------
  // Handle Delete
  // ---------------------------------------------------------

  const handleDeleteConfirm = async () => {
    if (!deleteConfirmDept) return;

    setIsSubmitting(true);

    try {
      await departmentApi.delete(
        deleteConfirmDept.id
      );

      setDeleteConfirmDept(null);

      onDataChanged?.();
    } catch (err: unknown) {
      if (err instanceof ApiClientError) {
        alert(
          `Failed to delete department: ${err.message}`
        );
      } else {
        alert(
          'Failed to delete department.'
        );
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  // ---------------------------------------------------------
  // Quick statistics
  // ---------------------------------------------------------

  const totalAssignedMembers =
    departments.reduce(
      (acc, department) =>
        acc + getDeptHeadcount(department.id),
      0
    );

  // ---------------------------------------------------------
  // Render
  // ---------------------------------------------------------

  return (
    <div
      className="fade-in"
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: '24px',
      }}
    >
      {/* =====================================================
          1. HEADER & SUMMARY STATS
      ====================================================== */}

      <div
        style={{
          display: 'grid',
          gridTemplateColumns:
            'repeat(auto-fit, minmax(220px, 1fr))',
          gap: '16px',
        }}
      >
        {/* Total Departments */}

        <div
          className="glass-panel"
          style={{
            padding: '20px',
            display: 'flex',
            alignItems: 'center',
            gap: '16px',
          }}
        >
          <div
            style={{
              width: '44px',
              height: '44px',
              borderRadius: '12px',
              background:
                'rgba(6, 182, 212, 0.12)',
              border:
                '1px solid rgba(6, 182, 212, 0.3)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'var(--secondary)',
            }}
          >
            <Building2 size={22} />
          </div>

          <div>
            <div
              style={{
                fontSize: '0.8125rem',
                color: 'var(--text-secondary)',
                fontWeight: 500,
              }}
            >
              Total Departments
            </div>

            <div
              style={{
                fontSize: '1.75rem',
                fontWeight: 700,
                fontFamily: 'var(--font-heading)',
              }}
            >
              {departments.length}
            </div>
          </div>
        </div>

        {/* Assigned Personnel */}

        <div
          className="glass-panel"
          style={{
            padding: '20px',
            display: 'flex',
            alignItems: 'center',
            gap: '16px',
          }}
        >
          <div
            style={{
              width: '44px',
              height: '44px',
              borderRadius: '12px',
              background:
                'var(--primary-subtle)',
              border:
                '1px solid rgba(99, 102, 241, 0.3)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: '#818cf8',
            }}
          >
            <Users size={22} />
          </div>

          <div>
            <div
              style={{
                fontSize: '0.8125rem',
                color: 'var(--text-secondary)',
                fontWeight: 500,
              }}
            >
              Assigned Personnel
            </div>

            <div
              style={{
                fontSize: '1.75rem',
                fontWeight: 700,
                fontFamily: 'var(--font-heading)',
              }}
            >
              {totalAssignedMembers}
            </div>
          </div>
        </div>

        {/* Relational Invariant */}

        <div
          className="glass-panel"
          style={{
            padding: '20px',
            display: 'flex',
            alignItems: 'center',
            gap: '16px',
          }}
        >
          <div
            style={{
              width: '44px',
              height: '44px',
              borderRadius: '12px',
              background:
                'rgba(16, 185, 129, 0.12)',
              border:
                '1px solid rgba(16, 185, 129, 0.3)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'var(--emerald)',
            }}
          >
            <Layers size={22} />
          </div>

          <div>
            <div
              style={{
                fontSize: '0.8125rem',
                color: 'var(--text-secondary)',
                fontWeight: 500,
              }}
            >
              Relational Invariant
            </div>

            <div
              style={{
                fontSize: '0.9rem',
                fontWeight: 600,
                color: 'var(--text-primary)',
              }}
            >
              ON DELETE SET NULL
            </div>
          </div>
        </div>
      </div>

      {/* =====================================================
          2. SEARCH & ACTION CONTROLS
      ====================================================== */}

      <div
        className="glass-panel"
        style={{
          padding: '20px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          flexWrap: 'wrap',
          gap: '16px',
        }}
      >
        {/* Search */}

        <div
          style={{
            position: 'relative',
            flex: '1 1 300px',
          }}
        >
          <Search
            size={16}
            style={{
              position: 'absolute',
              left: '14px',
              top: '50%',
              transform:
                'translateY(-50%)',
              color: 'var(--text-muted)',
            }}
          />

          <input
            type="text"
            className="input-field"
            placeholder="Search departments by name or description..."
            value={search}
            onChange={(e) =>
              setSearch(e.target.value)
            }
            style={{
              paddingLeft: '40px',
            }}
          />
        </div>

        {/* Actions */}

        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '12px',
          }}
        >
          {search && (
            <button
              onClick={() => setSearch('')}
              className="btn btn-ghost"
              style={{
                fontSize: '0.8rem',
              }}
            >
              <RotateCcw size={14} />

              <span>Reset</span>
            </button>
          )}

          {canManage ? (
            <button
              onClick={openCreateModal}
              className="btn btn-primary"
              style={{
                padding: '10px 18px',
              }}
            >
              <Plus size={16} />

              <span>
                Create Department
              </span>
            </button>
          ) : (
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
                color: 'var(--text-muted)',
                fontSize: '0.8rem',
              }}
            >
              <Shield size={14} />

              <span>
                Admin Role Required to Modify
              </span>
            </div>
          )}
        </div>
      </div>

      {/* =====================================================
          3. DEPARTMENT CARDS
      ====================================================== */}

      {filteredDepartments.length === 0 ? (
        <div
          className="glass-panel"
          style={{
            padding: '48px',
            textAlign: 'center',
            color: 'var(--text-muted)',
          }}
        >
          <Building2
            size={36}
            style={{
              margin: '0 auto 12px',
              opacity: 0.4,
            }}
          />

          <p
            style={{
              fontSize: '1rem',
              fontWeight: 500,
            }}
          >
            No departments found
          </p>

          <p
            style={{
              fontSize: '0.8125rem',
            }}
          >
            Try adjusting your search query or
            create a new department.
          </p>
        </div>
      ) : (
        <div
          style={{
            display: 'grid',
            gridTemplateColumns:
              'repeat(auto-fill, minmax(320px, 1fr))',
            gap: '20px',
          }}
        >
          {filteredDepartments.map(
            (department) => {
              const headcount =
                getDeptHeadcount(
                  department.id
                );

              const deptUsers =
                getDeptUsers(
                  department.id
                );

              return (
                <div
                  key={department.id}
                  className="glass-panel glass-panel-hover"
                  style={{
                    padding: '24px',
                    display: 'flex',
                    flexDirection: 'column',
                    justifyContent: 'space-between',
                    gap: '20px',
                    background:
                      'rgba(18, 24, 38, 0.7)',
                  }}
                >
                  <div>
                    {/* Card Header */}

                    <div
                      style={{
                        display: 'flex',
                        alignItems:
                          'flex-start',
                        justifyContent:
                          'space-between',
                        marginBottom: '12px',
                      }}
                    >
                      <div
                        style={{
                          display: 'flex',
                          alignItems:
                            'center',
                          gap: '12px',
                        }}
                      >
                        <div
                          style={{
                            width: '40px',
                            height: '40px',
                            borderRadius: '10px',
                            background:
                              'rgba(6, 182, 212, 0.14)',
                            border:
                              '1px solid rgba(6, 182, 212, 0.3)',
                            display: 'flex',
                            alignItems:
                              'center',
                            justifyContent:
                              'center',
                            color:
                              'var(--secondary)',
                          }}
                        >
                          <Building2
                            size={20}
                          />
                        </div>

                        <div>
                          <h4
                            style={{
                              fontSize:
                                '1.125rem',
                              fontWeight: 700,
                              color:
                                'var(--text-primary)',
                              margin:
                                '0 0 4px',
                            }}
                          >
                            {department.name}
                          </h4>

                          <span
                            style={{
                              fontSize:
                                '0.75rem',
                              color:
                                'var(--text-muted)',
                              fontFamily:
                                'monospace',
                            }}
                          >
                            ID: #
                            {department.id}
                          </span>
                        </div>
                      </div>

                      <span
                        className="badge badge-dept"
                        style={{
                          fontSize:
                            '0.75rem',
                        }}
                      >
                        {headcount}{' '}
                        {headcount === 1
                          ? 'member'
                          : 'members'}
                      </span>
                    </div>

                    {/* Description */}

                    <p
                      style={{
                        fontSize:
                          '0.85rem',
                        color:
                          'var(--text-secondary)',
                        lineHeight: 1.6,
                        minHeight: '42px',
                      }}
                    >
                      {department.description ||
                        (
                          <span
                            style={{
                              color:
                                'var(--text-muted)',
                              fontStyle:
                                'italic',
                            }}
                          >
                            No description
                            provided.
                          </span>
                        )}
                    </p>
                  </div>

                  {/* Card Footer */}

                  <div
                    style={{
                      paddingTop: '16px',
                      borderTop:
                        '1px solid var(--border-subtle)',
                      display: 'flex',
                      alignItems:
                        'center',
                      justifyContent:
                        'space-between',
                    }}
                  >
                    {/* Member preview */}

                    <div
                      style={{
                        display: 'flex',
                        alignItems:
                          'center',
                        gap: '-6px',
                      }}
                    >
                      {deptUsers
                        .slice(0, 3)
                        .map(
                          (user, index) => (
                            <div
                              key={user.id}
                              title={user.name}
                              style={{
                                width: '26px',
                                height: '26px',
                                borderRadius:
                                  '50%',
                                background:
                                  'linear-gradient(135deg, #6366f1, #06b6d4)',
                                color: '#fff',
                                fontSize:
                                  '0.7rem',
                                fontWeight: 700,
                                display:
                                  'flex',
                                alignItems:
                                  'center',
                                justifyContent:
                                  'center',
                                border:
                                  '2px solid var(--bg-surface)',
                                marginLeft:
                                  index > 0
                                    ? '-8px'
                                    : 0,
                              }}
                            >
                              {user.name
                                .charAt(0)
                                .toUpperCase()}
                            </div>
                          )
                        )}

                      {headcount > 3 && (
                        <span
                          style={{
                            fontSize:
                              '0.75rem',
                            color:
                              'var(--text-muted)',
                            marginLeft:
                              '6px',
                          }}
                        >
                          +{headcount - 3}{' '}
                          more
                        </span>
                      )}

                      {headcount === 0 && (
                        <span
                          style={{
                            fontSize:
                              '0.75rem',
                            color:
                              'var(--text-muted)',
                          }}
                        >
                          No members
                          assigned
                        </span>
                      )}
                    </div>

                    {/* Admin Actions */}

                    {canManage && (
                      <div
                        style={{
                          display:
                            'flex',
                          alignItems:
                            'center',
                          gap: '6px',
                        }}
                      >
                        <button
                          onClick={() =>
                            openEditModal(
                              department
                            )
                          }
                          className="btn btn-ghost"
                          style={{
                            padding:
                              '6px 10px',
                            color:
                              'var(--text-secondary)',
                          }}
                          title="Edit Department"
                        >
                          <Edit2
                            size={15}
                          />
                        </button>

                        <button
                          onClick={() =>
                            setDeleteConfirmDept(
                              department
                            )
                          }
                          className="btn btn-ghost"
                          style={{
                            padding:
                              '6px 10px',
                            color:
                              'var(--rose)',
                          }}
                          title="Delete Department"
                        >
                          <Trash2
                            size={15}
                          />
                        </button>
                      </div>
                    )}
                  </div>
                </div>
              );
            }
          )}
        </div>
      )}

      {/* =====================================================
          MODAL 1: CREATE / EDIT DEPARTMENT
      ====================================================== */}

      {isFormModalOpen && (
        <div
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor:
              'rgba(0, 0, 0, 0.75)',
            backdropFilter: 'blur(8px)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '20px',
            zIndex: 9999,
          }}
        >
          <div
            className="glass-panel fade-in"
            style={{
              width: '100%',
              maxWidth: '480px',
              padding: '28px',
            }}
          >
            {/* Modal Header */}

            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent:
                  'space-between',
                marginBottom: '20px',
              }}
            >
              <div
                style={{
                  display: 'flex',
                  alignItems:
                    'center',
                  gap: '10px',
                }}
              >
                <Building2
                  size={20}
                  color="var(--secondary)"
                />

                <h3
                  style={{
                    fontSize:
                      '1.25rem',
                    fontWeight: 700,
                    margin: 0,
                  }}
                >
                  {editingDept
                    ? `Edit Department #${editingDept.id}`
                    : 'Create New Department'}
                </h3>
              </div>

              <button
                type="button"
                onClick={
                  closeFormModal
                }
                className="btn btn-ghost"
                style={{
                  padding: '6px',
                }}
                disabled={
                  isSubmitting
                }
              >
                <X size={18} />
              </button>
            </div>

            {/* Form Error */}

            {formError && (
              <div
                style={{
                  background:
                    'var(--rose-bg)',
                  border:
                    '1px solid var(--rose-border)',
                  borderRadius:
                    'var(--radius-md)',
                  padding:
                    '12px 14px',
                  marginBottom:
                    '16px',
                  color:
                    'var(--rose)',
                  fontSize:
                    '0.85rem',
                }}
              >
                <div
                  style={{
                    display:
                      'flex',
                    alignItems:
                      'center',
                    gap: '8px',
                  }}
                >
                  <AlertCircle
                    size={16}
                  />

                  <span>
                    {formError}
                  </span>
                </div>

                {formRequestId && (
                  <div
                    style={{
                      fontSize:
                        '0.7rem',
                      color:
                        'var(--text-muted)',
                      fontFamily:
                        'monospace',
                      marginTop:
                        '4px',
                    }}
                  >
                    Trace ID:{' '}
                    {formRequestId}
                  </div>
                )}
              </div>
            )}

            {/* Form */}

            <form
              onSubmit={
                handleFormSubmit
              }
            >
              {/* Department Name */}

              <div className="input-group">
                <label
                  className="input-label"
                  htmlFor="deptName"
                >
                  Department Name *
                </label>

                <input
                  id="deptName"
                  type="text"
                  className="input-field"
                  placeholder="e.g. Platform Engineering"
                  value={formName}
                  onChange={(e) =>
                    setFormName(
                      e.target.value
                    )
                  }
                  required
                  disabled={
                    isSubmitting
                  }
                />
              </div>

              {/* Description */}

              <div className="input-group">
                <label
                  className="input-label"
                  htmlFor="deptDesc"
                >
                  Description
                </label>

                <textarea
                  id="deptDesc"
                  className="input-field"
                  rows={3}
                  placeholder="Brief description of department scope and team responsibilities..."
                  value={
                    formDescription
                  }
                  onChange={(e) =>
                    setFormDescription(
                      e.target.value
                    )
                  }
                  style={{
                    resize:
                      'vertical',
                  }}
                  disabled={
                    isSubmitting
                  }
                />
              </div>

              {/* Buttons */}

              <div
                style={{
                  display:
                    'flex',
                  justifyContent:
                    'flex-end',
                  gap: '10px',
                  marginTop:
                    '24px',
                }}
              >
                <button
                  type="button"
                  onClick={
                    closeFormModal
                  }
                  className="btn btn-secondary"
                  disabled={
                    isSubmitting
                  }
                >
                  Cancel
                </button>

                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={
                    isSubmitting
                  }
                >
                  {isSubmitting
                    ? 'Saving...'
                    : editingDept
                      ? 'Update Department'
                      : 'Create Department'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* =====================================================
          MODAL 2: DELETE CONFIRMATION
      ====================================================== */}

      {deleteConfirmDept && (
        <div
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor:
              'rgba(0, 0, 0, 0.75)',
            backdropFilter: 'blur(8px)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '20px',
            zIndex: 9999,
          }}
        >
          <div
            className="glass-panel fade-in"
            style={{
              width: '100%',
              maxWidth: '440px',
              padding: '28px',
            }}
          >
            {/* Delete Header */}

            <div
              style={{
                display: 'flex',
                alignItems:
                  'center',
                gap: '12px',
                marginBottom:
                  '16px',
                color:
                  'var(--rose)',
              }}
            >
              <div
                style={{
                  padding: '8px',
                  borderRadius:
                    '10px',
                  background:
                    'var(--rose-bg)',
                }}
              >
                <Trash2
                  size={24}
                />
              </div>

              <h3
                style={{
                  fontSize:
                    '1.2rem',
                  fontWeight: 700,
                  color:
                    'var(--text-primary)',
                  margin: 0,
                }}
              >
                Delete Department
              </h3>
            </div>

            {/* Delete Question */}

            <p
              style={{
                color:
                  'var(--text-secondary)',
                fontSize:
                  '0.875rem',
                lineHeight: 1.6,
                marginBottom:
                  '16px',
              }}
            >
              Are you sure you want
              to delete{' '}
              <strong>
                {deleteConfirmDept.name}
              </strong>
              ?
            </p>

            {/* Relational Safety Notice */}

            <div
              style={{
                background:
                  'rgba(245, 158, 11, 0.1)',
                border:
                  '1px solid rgba(245, 158, 11, 0.3)',
                borderRadius:
                  'var(--radius-md)',
                padding:
                  '12px 14px',
                marginBottom:
                  '20px',
                fontSize:
                  '0.8125rem',
                color:
                  'var(--amber)',
                display:
                  'flex',
                alignItems:
                  'flex-start',
                gap: '10px',
              }}
            >
              <AlertCircle
                size={18}
                style={{
                  flexShrink: 0,
                  marginTop:
                    '2px',
                }}
              />

              <div>
                <strong>
                  Relational Integrity
                  Guard:
                </strong>

                <p
                  style={{
                    marginTop:
                      '4px',
                    lineHeight: 1.5,
                  }}
                >
                  {getDeptHeadcount(
                    deleteConfirmDept.id
                  ) > 0 ? (
                    <>
                      <strong>
                        {getDeptHeadcount(
                          deleteConfirmDept.id
                        )}{' '}
                        user(s)
                      </strong>{' '}
                      are currently
                      assigned to this
                      department.
                      Deleting it
                      triggers{' '}
                      <code
                        style={{
                          color: '#fff',
                          background:
                            'rgba(0,0,0,0.3)',
                          padding:
                            '2px 4px',
                          borderRadius:
                            '4px',
                        }}
                      >
                        ON DELETE SET NULL
                      </code>
                      , unassigning those
                      users without
                      deleting their
                      accounts.
                    </>
                  ) : (
                    <>
                      No users are
                      currently assigned
                      to this department.
                      The record will be
                      cleanly removed.
                    </>
                  )}
                </p>
              </div>
            </div>

            {/* Delete Buttons */}

            <div
              style={{
                display: 'flex',
                justifyContent:
                  'flex-end',
                gap: '10px',
              }}
            >
              <button
                type="button"
                onClick={() =>
                  setDeleteConfirmDept(
                    null
                  )
                }
                className="btn btn-secondary"
                disabled={
                  isSubmitting
                }
              >
                Cancel
              </button>

              <button
                type="button"
                onClick={
                  handleDeleteConfirm
                }
                className="btn btn-danger"
                disabled={
                  isSubmitting
                }
              >
                {isSubmitting
                  ? 'Deleting...'
                  : 'Delete Department'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default DepartmentsView;