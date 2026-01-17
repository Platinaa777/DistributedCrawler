import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { API_CONFIG, API_ENDPOINTS } from '../../constants/api.constants';
import { ListUsersResponse, UpdateUserRoleResponse, UserRole } from '../../models';

@Injectable({
  providedIn: 'root'
})
export class UserApiService {
  private readonly baseUrl = `${API_CONFIG.BASE_URL}${API_CONFIG.API_PREFIX}`;

  constructor(private http: HttpClient) {}

  listUsers(): Observable<ListUsersResponse> {
    return this.http.get<ListUsersResponse>(`${this.baseUrl}${API_ENDPOINTS.USERS}`);
  }

  updateUserRole(id: string, role: UserRole): Observable<UpdateUserRoleResponse> {
    return this.http.patch<UpdateUserRoleResponse>(
      `${this.baseUrl}${API_ENDPOINTS.USERS}/${id}/role`,
      { id, role }
    );
  }
}
