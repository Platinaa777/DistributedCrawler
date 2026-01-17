export type UserRole = 'READ' | 'READ_WRITE' | 'ADMINISTRATOR';

export interface User {
  id: string;
  email: string;
  role: UserRole;
  created_at: string;
  updated_at: string;
}

export interface ListUsersResponse {
  users: User[];
}

export interface UpdateUserRoleResponse {
  updated: boolean;
}
