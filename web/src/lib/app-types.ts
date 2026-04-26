export type TabKey =
  | "overview"
  | "machines"
  | "sshKeys"
  | "thresholds"
  | "notifications"
  | "samples"
  | "alerts";

export type LoginFormState = {
  username: string;
  password: string;
};

export type SSHKeyImportState = {
  name: string;
  privateKey: string;
};

export type SSHKeyGenerateState = {
  name: string;
};

export type MachineFormState = {
  name: string;
  host: string;
  port: string;
  sshUser: string;
  networkInterface: string;
  sshKeyID: string;
  collectEnabled: boolean;
  remark: string;
};

export type ThresholdFormRow = {
  period_type: string;
  metric_type: string;
  threshold_value: string;
  threshold_unit: "MB" | "GB";
  enabled: boolean;
  source?: string;
};

export type WebhookFormState = {
  enabled: boolean;
  method: "GET" | "POST";
  url: string;
  headersText: string;
  bodyText: string;
  proxyID: string;
};

export type WebhookPreviewState = {
  url: string;
  headersText: string;
  bodyText: string;
};

export type TelegramPreviewState = {
  messageText: string;
};

export type TelegramFormState = {
  enabled: boolean;
  botToken: string;
  chatID: string;
  messageText: string;
  proxyID: string;
};

export type NotificationProxyFormState = {
  id: number | null;
  name: string;
  proxyType: "http" | "socks";
  url: string;
};

export type ThresholdRulePayload = {
  period_type: string;
  metric_type: string;
  threshold_value: number;
  threshold_unit: "MB" | "GB";
  enabled: boolean;
};

export type TabDefinition = {
  key: TabKey;
  label: string;
  path: string;
};

export type MachineOption = {
  value: number;
  label: string;
};
