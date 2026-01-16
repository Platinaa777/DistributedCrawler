import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { CrawlJob, CrawlJobConfig, CrawlTask, FileType, JobExportFileType, JobExportFileURLResponse, TaskFileURLResponse } from '../../models';
import { API_CONFIG, API_ENDPOINTS } from '../../constants/api.constants';

// Filter options for listing jobs
export interface JobListFilter {
  name?: string;
  status?: string;
  created_from?: string; // ISO 8601 timestamp
  created_to?: string;   // ISO 8601 timestamp
}

// Parameters for listing jobs with cursor pagination
export interface JobListParams {
  cursor?: string;
  limit?: number;
  filter?: JobListFilter;
}

// Response from list jobs API with cursor pagination
export interface JobListResponse {
  jobs: CrawlJob[];
  next_cursor: string;
  has_more: boolean;
}

@Injectable({
  providedIn: 'root'
})
export class CrawlerApiService {
  private readonly baseUrl = `${API_CONFIG.BASE_URL}${API_CONFIG.API_PREFIX}`;

  constructor(private http: HttpClient) {}

  // Jobs
  listJobs(params?: JobListParams): Observable<JobListResponse> {
    let httpParams = new HttpParams();

    if (params?.cursor) {
      httpParams = httpParams.set('cursor', params.cursor);
    }
    if (params?.limit) {
      httpParams = httpParams.set('limit', params.limit.toString());
    }

    // Add filter params
    if (params?.filter) {
      if (params.filter.name) {
        httpParams = httpParams.set('filter.name', params.filter.name);
      }
      if (params.filter.status) {
        httpParams = httpParams.set('filter.status', params.filter.status);
      }
      if (params.filter.created_from) {
        httpParams = httpParams.set('filter.created_from', params.filter.created_from);
      }
      if (params.filter.created_to) {
        httpParams = httpParams.set('filter.created_to', params.filter.created_to);
      }
    }

    return this.http.get<JobListResponse>(`${this.baseUrl}${API_ENDPOINTS.JOBS}`, { params: httpParams });
  }

  getJob(id: string): Observable<{ job: CrawlJob }> {
    return this.http.get<{ job: CrawlJob }>(`${this.baseUrl}${API_ENDPOINTS.JOBS}/${id}`);
  }

  getJobExportFileURL(jobId: string, fileType: JobExportFileType): Observable<JobExportFileURLResponse> {
    const params = new HttpParams().set('file_type', fileType);
    return this.http.get<JobExportFileURLResponse>(
      `${this.baseUrl}${API_ENDPOINTS.JOBS}/${jobId}/export-url`,
      { params }
    );
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

  // File URLs
  getTaskFileURL(taskId: string, fileType: FileType): Observable<TaskFileURLResponse> {
    let params = new HttpParams().set('file_type', fileType);
    return this.http.get<TaskFileURLResponse>(
      `${this.baseUrl}${API_ENDPOINTS.TASKS}/${taskId}/file-url`,
      { params }
    );
  }
}
