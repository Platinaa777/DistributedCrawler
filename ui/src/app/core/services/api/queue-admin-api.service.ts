import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { API_CONFIG, API_ENDPOINTS } from '../../constants/api.constants';
import {
  QueueEndpoint,
  QueueRoutingRule,
  ListQueueEndpointsResponse,
  ListQueueRoutingRulesResponse,
  QueueStage
} from '../../models/queue.model';

@Injectable({
  providedIn: 'root'
})
export class QueueAdminApiService {
  private readonly baseUrl = `${API_CONFIG.BASE_URL}${API_CONFIG.API_PREFIX}`;

  constructor(private http: HttpClient) {}

  listEndpoints(): Observable<ListQueueEndpointsResponse> {
    return this.http.get<ListQueueEndpointsResponse>(`${this.baseUrl}${API_ENDPOINTS.QUEUES}`);
  }

  createEndpoint(endpoint: Partial<QueueEndpoint>): Observable<{ endpoint: QueueEndpoint }> {
    return this.http.post<{ endpoint: QueueEndpoint }>(`${this.baseUrl}${API_ENDPOINTS.QUEUES}`, endpoint);
  }

  updateEndpoint(endpoint: QueueEndpoint): Observable<{ endpoint: QueueEndpoint }> {
    return this.http.patch<{ endpoint: QueueEndpoint }>(
      `${this.baseUrl}${API_ENDPOINTS.QUEUES}/${endpoint.id}`,
      endpoint
    );
  }

  deleteEndpoint(id: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}${API_ENDPOINTS.QUEUES}/${id}`);
  }

  listRoutingRules(stage?: QueueStage): Observable<ListQueueRoutingRulesResponse> {
    const params: Record<string, string> = {};
    if (stage) params['stage'] = stage;
    return this.http.get<ListQueueRoutingRulesResponse>(`${this.baseUrl}${API_ENDPOINTS.QUEUE_ROUTING}`, { params });
  }

  upsertRoutingRule(rule: Partial<QueueRoutingRule>): Observable<{ rule: QueueRoutingRule }> {
    return this.http.put<{ rule: QueueRoutingRule }>(`${this.baseUrl}${API_ENDPOINTS.QUEUE_ROUTING}`, rule);
  }
}
