export interface Department {
  id: number;
  name: string;
  description: string;
  created_at?: string;
  updated_at?: string;
}

export interface Role {
  id: number;
  name: string;
  description: string;
  created_at?: string;
  updated_at?: string;
}

export interface User {
  id: number;
  name: string;
  email: string;
  department_id?: number | null;
  status: 'active' | 'inactive';
  created_at: string;
  updated_at: string;
  department?: Department | null;
  roles: Role[];
}

export interface PaginationMeta {
  total_count: number;
  page: number;
  limit: number;
  total_pages: number;
}

export interface ApiError {
  code: string;
  message: string;
  details?: unknown;
  request_id?: string;
}

export interface ApiResponse<T> {
  success: boolean;
  data: T;
  pagination?: PaginationMeta;
  error?: ApiError;
}

export interface LoginResponse {
  token: string;
  user: User;
  roles: string[];
}

export interface UserFilterQuery {
  page?: number;
  limit?: number;
  search?: string;
  department_id?: number;
  role_id?: number;
  status?: string;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
}

export interface CreateUserPayload {
  name: string;
  email: string;
  password?: string;
  department_id?: number | null;
  role_ids: number[];
  status: 'active' | 'inactive';
}

export interface UpdateUserPayload {
  name?: string;
  email?: string;
  department_id?: number | null;
  role_ids?: number[];
  status?: 'active' | 'inactive';
}
