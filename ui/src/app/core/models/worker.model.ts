export interface WorkerInfo {
  worker_id: string;
  worker_type?: string;
  status: string;
  last_heartbeat_at?: string;
  active_tasks: number;
  uptime?: string;
}

export interface ListWorkersResponse {
  workers: WorkerInfo[];
}
