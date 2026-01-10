import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { CrawlJob, CrawlJobConfig, CrawlTask } from '../../models';
import { API_CONFIG, API_ENDPOINTS } from '../../constants/api.constants';

@Injectable({
  providedIn: 'root'
})
export class CrawlerApiService {
  private readonly baseUrl = `${API_CONFIG.BASE_URL}${API_CONFIG.API_PREFIX}`;

  constructor(private http: HttpClient) {}

  // Jobs
  listJobs(params?: { status?: string; limit?: number; offset?: number }): Observable<{ jobs: CrawlJob[] }> {
    let httpParams = new HttpParams();
    if (params?.status) {
      httpParams = httpParams.set('status', params.status);
    }
    if (params?.limit) {
      httpParams = httpParams.set('limit', params.limit.toString());
    }
    if (params?.offset) {
      httpParams = httpParams.set('offset', params.offset.toString());
    }

    return this.http.get<{ jobs: CrawlJob[] }>(`${this.baseUrl}${API_ENDPOINTS.JOBS}`, { params: httpParams });
  }

  getJob(id: string): Observable<{ job: CrawlJob }> {
    return this.http.get<{ job: CrawlJob }>(`${this.baseUrl}${API_ENDPOINTS.JOBS}/${id}`);
  }

  createJob(config: CrawlJobConfig): Observable<{ id: string }> {
    return this.http.post<{ id: string }>(`${this.baseUrl}${API_ENDPOINTS.JOBS}`, { config });
  }

  // Tasks
  listTasksByJob(jobId: string): Observable<{ tasks: CrawlTask[] }> {
    return this.http.get<{ tasks: CrawlTask[] }>(`${this.baseUrl}${API_ENDPOINTS.JOBS}/${jobId}/tasks`);
  }

  getTask(id: string): Observable<{ task: CrawlTask }> {
    return this.http.get<{ task: CrawlTask }>(`${this.baseUrl}${API_ENDPOINTS.TASKS}/${id}`);
  }
}
