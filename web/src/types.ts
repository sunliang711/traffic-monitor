export type AdminProfile = {
  id: number;
  username: string;
};

export type SSHKey = {
  id: number;
  name: string;
  source_type: string;
  key_type: string;
  public_key: string;
  fingerprint: string;
  created_at: string;
  updated_at: string;
};

export type Machine = {
  id: number;
  name: string;
  host: string;
  port: number;
  ssh_user: string;
  network_interface: string;
  ssh_key_id: number;
  collect_enabled: boolean;
  remark: string;
  created_at: string;
  updated_at: string;
};

export type ThresholdRule = {
  period_type: string;
  metric_type: string;
  threshold_mb: number;
  threshold_value: number;
  threshold_unit: string;
  enabled: boolean;
  source?: string;
};

export type NotificationChannel = {
  channel_type: string;
  enabled: boolean;
  configured: boolean;
  url?: string;
  method?: "GET" | "POST";
  headers?: Record<string, string>;
  body?: string;
  chat_id?: string;
  token_masked?: string;
};

export type TrafficSample = {
  id: number;
  machine_id: number;
  period_type: string;
  bucket_time: string;
  upload_mb: number;
  download_mb: number;
  total_mb: number;
  collected_at: string;
};

export type TrafficSampleList = {
  items: TrafficSample[];
  total: number;
};

export type AlertItem = {
  id: number;
  machine_id: number;
  period_type: string;
  metric_type: string;
  bucket_time: string;
  threshold_mb: number;
  actual_mb: number;
  notify_status: string;
  notified_at?: string;
  created_at: string;
};

export type AlertList = {
  items: AlertItem[];
  total: number;
};

export type CollectNowResult = {
  machine_id: number;
  status: string;
  sample_count: number;
  error?: string;
};

export type CollectNowResponse = {
  results: CollectNowResult[];
};

export type ConnectionTestResponse = {
  machine_id: number;
  ssh_reachable: boolean;
  vnstat_ready: boolean;
  vnstat_version: string;
  status: string;
};

export type WebhookTestResponse = {
  status_code: number;
  body: string;
  rendered_url?: string;
  rendered_headers?: Record<string, string>;
  rendered_body?: string;
};

export type CleanupHistoryResponse = {
  deleted_samples: number;
  deleted_alerts: number;
  samples_cutoff: string;
  alerts_cutoff: string;
};
