import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { CrawlJob, CrawlJobConfig, CrawlTask, FileType, JobExportFileType, JobExportFileURLResponse, ListWorkersResponse, TaskFileURLResponse } from '../../models';
import { API_CONFIG, API_ENDPOINTS } from '../../constants/api.constants';

// Filter options for listing jobs
export interface JobListFilter {
  name?: string;
  user_email?: string;
  status?: string;
  created_from?: string; // ISO 8601 timestamp
  created_to?: string;   // ISO 8601 timestamp
}

// Sort options for listing jobs
export interface JobSortParams {
  sort_field?: 'JOB_SORT_FIELD_CREATED_AT' | 'JOB_SORT_FIELD_NAME' | 'JOB_SORT_FIELD_STATUS';
  sort_order?: 'SORT_ORDER_ASC' | 'SORT_ORDER_DESC';
}

// Parameters for listing jobs with cursor pagination
export interface JobListParams {
  cursor?: string;
  limit?: number;
  filter?: JobListFilter;
  sort?: JobSortParams;
}

// Response from list jobs API with cursor pagination
export interface JobListResponse {
  jobs: CrawlJob[];
  next_cursor: string;
  has_more: boolean;
}

// Filter options for listing tasks
export interface TaskListFilter {
  status?: string;
  url?: string;
  min_depth?: number;
  max_depth?: number;
  enqueued_from?: string; // ISO 8601 timestamp
  enqueued_to?: string;   // ISO 8601 timestamp
}

// Sort options for listing tasks
export interface TaskSortParams {
  sort_field?: 'TASK_SORT_FIELD_ENQUEUED_AT' | 'TASK_SORT_FIELD_URL' | 'TASK_SORT_FIELD_STATUS' | 'TASK_SORT_FIELD_DEPTH';
  sort_order?: 'SORT_ORDER_ASC' | 'SORT_ORDER_DESC';
}

// Parameters for listing tasks with cursor pagination
export interface TaskListParams {
  cursor?: string;
  limit?: number;
  filter?: TaskListFilter;
  sort?: TaskSortParams;
}

// Response from list tasks API with cursor pagination
export interface TaskListResponse {
  tasks: CrawlTask[];
  next_cursor: string;
  has_more: boolean;
}

// Task analytics response
export interface TaskAnalytics {
  status_counts: { [status: string]: number };
  depth_counts: { [depth: string]: number };
  total_count: number;
}

export interface TaskAnalyticsResponse {
  analytics: TaskAnalytics;
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
      if (params.filter.user_email) {
        httpParams = httpParams.set('filter.user_email', params.filter.user_email);
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

    // Add sort params
    if (params?.sort?.sort_field) {
      httpParams = httpParams.set('sort_field', params.sort.sort_field);
    }
    if (params?.sort?.sort_order) {
      httpParams = httpParams.set('sort_order', params.sort.sort_order);
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

  deleteJob(id: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}${API_ENDPOINTS.JOBS}/${id}`);
  }

  // Tasks - Updated with pagination and filtering
  listTasksByJob(jobId: string, params?: TaskListParams): Observable<TaskListResponse> {
    let httpParams = new HttpParams();

    if (params?.cursor) {
      httpParams = httpParams.set('cursor', params.cursor);
    }
    if (params?.limit) {
      httpParams = httpParams.set('limit', params.limit.toString());
    }

    // Add filter params
    if (params?.filter) {
      if (params.filter.status) {
        httpParams = httpParams.set('filter.status', params.filter.status);
      }
      if (params.filter.url) {
        httpParams = httpParams.set('filter.url', params.filter.url);
      }
      if (params.filter.min_depth !== undefined) {
        httpParams = httpParams.set('filter.min_depth', params.filter.min_depth.toString());
      }
      if (params.filter.max_depth !== undefined) {
        httpParams = httpParams.set('filter.max_depth', params.filter.max_depth.toString());
      }
      if (params.filter.enqueued_from) {
        httpParams = httpParams.set('filter.enqueued_from', params.filter.enqueued_from);
      }
      if (params.filter.enqueued_to) {
        httpParams = httpParams.set('filter.enqueued_to', params.filter.enqueued_to);
      }
    }

    // Add sort params
    if (params?.sort?.sort_field) {
      httpParams = httpParams.set('sort_field', params.sort.sort_field);
    }
    if (params?.sort?.sort_order) {
      httpParams = httpParams.set('sort_order', params.sort.sort_order);
    }

    return this.http.get<TaskListResponse>(`${this.baseUrl}${API_ENDPOINTS.JOBS}/${jobId}/tasks`, { params: httpParams });
  }

  // Get task analytics for a job
  getTaskAnalytics(jobId: string): Observable<TaskAnalyticsResponse> {
    return this.http.get<TaskAnalyticsResponse>(`${this.baseUrl}${API_ENDPOINTS.JOBS}/${jobId}/analytics`);
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

  // Workers
  listWorkers(): Observable<ListWorkersResponse> {
    return this.http.get<ListWorkersResponse>(`${this.baseUrl}${API_ENDPOINTS.WORKERS}`);
  }
}
