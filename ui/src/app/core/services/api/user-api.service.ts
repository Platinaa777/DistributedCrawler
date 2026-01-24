import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { API_CONFIG, API_ENDPOINTS } from '../../constants/api.constants';
import {
  ApiListUsersResponse,
  ListUsersResponse,
  UpdateUserRoleResponse,
  UserRole,
  mapApiUser,
  toProtoRole
} from '../../models';

@Injectable({
  providedIn: 'root'
})
export class UserApiService {
  private readonly baseUrl = `${API_CONFIG.BASE_URL}${API_CONFIG.API_PREFIX}`;

  constructor(private http: HttpClient) {}

  listUsers(): Observable<ListUsersResponse> {
    return this.http.get<ApiListUsersResponse>(`${this.baseUrl}${API_ENDPOINTS.USERS}`).pipe(
      map(response => ({
        users: (response.users || []).map(mapApiUser)
      }))
    );
  }

  updateUserRole(id: string, role: UserRole): Observable<UpdateUserRoleResponse> {
    return this.http.patch<UpdateUserRoleResponse>(
      `${this.baseUrl}${API_ENDPOINTS.USERS}/${id}/role`,
      { role: toProtoRole(role) }
    );
  }
}
