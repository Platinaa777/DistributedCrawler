export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface AuthTokens {
  accessToken: string;
  refreshToken: string;
  expiresAt: number; // unix ms
}
