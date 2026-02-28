export type QueueStage = 'QUEUE_STAGE_UNSPECIFIED' | 'QUEUE_STAGE_CRAWL' | 'QUEUE_STAGE_PARSE';
export type QueueBrokerType = 'QUEUE_BROKER_TYPE_UNSPECIFIED' | 'QUEUE_BROKER_TYPE_RABBITMQ' | 'QUEUE_BROKER_TYPE_KAFKA';

export interface QueueEndpoint {
  id: string;
  display_name: string;
  broker_type: QueueBrokerType;
  stage: QueueStage;
  host: string;
  queue_name: string;
  secret_key: string;
  created_at: string;
  updated_at: string;
}

export interface QueueRoutingRule {
  id: string;
  stage: QueueStage;
  scope: string;
}

export interface ListQueueEndpointsResponse {
  endpoints: QueueEndpoint[];
}

export interface ListQueueRoutingRulesResponse {
  rules: QueueRoutingRule[];
}
