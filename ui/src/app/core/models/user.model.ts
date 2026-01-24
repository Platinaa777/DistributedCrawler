export type UserRole = 'READ' | 'READ_WRITE' | 'ADMINISTRATOR';
export type ProtoUserRole = 'ROLE_READ' | 'ROLE_READ_WRITE' | 'ROLE_ADMINISTRATOR';

export interface User {
  id: string;
  email: string;
  role: UserRole;
  created_at: string;
  updated_at: string;
}

// Raw API response with proto-style roles
export interface ApiUser {
  id: string;
  email: string;
  role: ProtoUserRole;
  created_at: string;
  updated_at: string;
}

export interface ListUsersResponse {
  users: User[];
}

export interface ApiListUsersResponse {
  users: ApiUser[];
}

export interface UpdateUserRoleResponse {
  updated: boolean;
}

// Convert proto role (ROLE_READ) to frontend role (READ)
export function fromProtoRole(protoRole: ProtoUserRole): UserRole {
  switch (protoRole) {
    case 'ROLE_READ':
      return 'READ';
    case 'ROLE_READ_WRITE':
      return 'READ_WRITE';
    case 'ROLE_ADMINISTRATOR':
      return 'ADMINISTRATOR';
    default:
      return 'READ';
  }
}

// Convert frontend role (READ) to proto role (ROLE_READ)
export function toProtoRole(role: UserRole): ProtoUserRole {
  return `ROLE_${role}` as ProtoUserRole;
}

// Convert API user to frontend user
export function mapApiUser(apiUser: ApiUser): User {
  return {
    ...apiUser,
    role: fromProtoRole(apiUser.role)
  };
}
