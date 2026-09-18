import type {
  ApiResponse,
  LoginResponse,
  User,
  Department,
  Role,
  UserFilterQuery,
  CreateUserPayload,
  UpdateUserPayload,
} from './types';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';
const HEALTH_URL = import.meta.env.VITE_HEALTH_URL || 'http://localhost:8080/health';

export class ApiClientError extends Error {
  code: string;
  statusCode: number;
  requestId?: string;
  details?: unknown;

  constructor(message: string, code = 'UNKNOWN_ERROR', statusCode = 500, requestId?: string, details?: unknown) {
    super(message);
    this.name = 'ApiClientError';
    this.code = code;
    this.statusCode = statusCode;
    this.requestId = requestId;
    this.details = details;
  }
}

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<ApiResponse<T>> {
  const token = localStorage.getItem('auth_token');
  const headers = new Headers(options.headers || {});

  if (!headers.has('Content-Type') && !(options.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json');
  }

  if (token && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  const url = endpoint.startsWith('http') ? endpoint : `${API_BASE_URL}${endpoint}`;

  try {
    const res = await fetch(url, {
      ...options,
      headers,
    });

    const requestId = res.headers.get('X-Request-ID') || undefined;

    let json: ApiResponse<T>;
    try {
      json = await res.json();
    } catch {
      throw new ApiClientError(
        `Failed to parse server response (HTTP ${res.status})`,
        'INVALID_JSON',
        res.status,
        requestId
      );
    }

    if (!res.ok || json.success === false) {
      const errCode = json.error?.code || `HTTP_${res.status}`;
      const errMsg = json.error?.message || `Request failed with status ${res.status}`;
      const errReqId = json.error?.request_id || requestId;
      throw new ApiClientError(errMsg, errCode, res.status, errReqId, json.error?.details);
    }

    return json;
  } catch (err: unknown) {
    if (err instanceof ApiClientError) {
      throw err;
    }
    const networkError = err as Error;
    throw new ApiClientError(
      networkError.message || 'Network connection failed. Is the API Gateway running on :8080?',
      'NETWORK_ERROR',
      0
    );
  }
}

// -------------------------------------------------------------
// Authentication API
// -------------------------------------------------------------
export const authApi = {
  login: async (email: string, password: string): Promise<LoginResponse> => {
    const res = await request<LoginResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
    return res.data;
  },
};

// -------------------------------------------------------------
// User Management API
// -------------------------------------------------------------
export const userApi = {
  list: async (query?: UserFilterQuery): Promise<{ users: User[]; pagination?: ApiResponse<User[]>['pagination'] }> => {
    const params = new URLSearchParams();
    if (query?.page) params.set('page', String(query.page));
    if (query?.limit) params.set('limit', String(query.limit));
    if (query?.search) params.set('search', query.search);
    if (query?.department_id) params.set('department_id', String(query.department_id));
    if (query?.role_id) params.set('role_id', String(query.role_id));
    if (query?.status) params.set('status', query.status);
    if (query?.sort_by) params.set('sort_by', query.sort_by);
    if (query?.sort_order) params.set('sort_order', query.sort_order);

    const qs = params.toString();
    const res = await request<User[]>(`/users${qs ? `?${qs}` : ''}`);
    return {
      users: res.data || [],
      pagination: res.pagination,
    };
  },

  get: async (id: number): Promise<User> => {
    const res = await request<User>(`/users/${id}`);
    return res.data;
  },

  create: async (payload: CreateUserPayload): Promise<User> => {
    const res = await request<User>('/users', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
    return res.data;
  },

  update: async (id: number, payload: UpdateUserPayload): Promise<User> => {
    const res = await request<User>(`/users/${id}`, {
      method: 'PUT',
      body: JSON.stringify(payload),
    });
    return res.data;
  },

  delete: async (id: number): Promise<void> => {
    await request(`/users/${id}`, {
      method: 'DELETE',
    });
  },
};

// -------------------------------------------------------------
// Department Management API
// -------------------------------------------------------------
export const departmentApi = {
  list: async (): Promise<Department[]> => {
    const res = await request<Department[]>('/departments');
    return res.data || [];
  },

  create: async (name: string, description: string): Promise<Department> => {
    const res = await request<Department>('/departments', {
      method: 'POST',
      body: JSON.stringify({ name, description }),
    });
    return res.data;
  },

  update: async (id: number, name: string, description: string): Promise<Department> => {
    const res = await request<Department>(`/departments/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ name, description }),
    });
    return res.data;
  },

  delete: async (id: number): Promise<void> => {
    await request(`/departments/${id}`, {
      method: 'DELETE',
    });
  },
};

// -------------------------------------------------------------
// Role Management API
// -------------------------------------------------------------
export const roleApi = {
  list: async (): Promise<Role[]> => {
    const res = await request<Role[]>('/roles');
    return res.data || [];
  },

  create: async (name: string, description: string): Promise<Role> => {
    const res = await request<Role>('/roles', {
      method: 'POST',
      body: JSON.stringify({ name, description }),
    });
    return res.data;
  },

  update: async (id: number, name: string, description: string): Promise<Role> => {
    const res = await request<Role>(`/roles/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ name, description }),
    });
    return res.data;
  },

  delete: async (id: number): Promise<void> => {
    await request(`/roles/${id}`, {
      method: 'DELETE',
    });
  },
};

// -------------------------------------------------------------
// Health API
// -------------------------------------------------------------
export const systemApi = {
  checkHealth: async (): Promise<boolean> => {
    try {
      const res = await fetch(HEALTH_URL);
      return res.ok;
    } catch {
      return false;
    }
  },
};
