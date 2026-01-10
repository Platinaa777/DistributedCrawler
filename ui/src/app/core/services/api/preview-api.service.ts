import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { Preview } from '../../models';
import { API_CONFIG, API_ENDPOINTS } from '../../constants/api.constants';

@Injectable({
  providedIn: 'root'
})
export class PreviewApiService {
  private readonly baseUrl = `${API_CONFIG.BASE_URL}${API_CONFIG.API_PREFIX}`;

  constructor(private http: HttpClient) {}

  createPreview(url: string): Observable<{ id: string }> {
    return this.http.post<{ id: string }>(`${this.baseUrl}${API_ENDPOINTS.PREVIEWS}`, { url });
  }

  getPreview(id: string): Observable<{ preview: Preview }> {
    return this.http.get<{ preview: Preview }>(`${this.baseUrl}${API_ENDPOINTS.PREVIEWS}/${id}`);
  }

  // Fetch raw HTML from presigned URL
  fetchPreviewHtml(downloadUrl: string): Observable<string> {
    return this.http.get(downloadUrl, { responseType: 'text' });
  }
}
