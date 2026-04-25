export type TabKey =
  | "overview"
  | "machines"
  | "sshKeys"
  | "thresholds"
  | "notifications"
  | "samples"
  | "alerts";

export type ThresholdFormRow = {
  period_type: string;
  metric_type: string;
  threshold_value: string;
  threshold_unit: "MB" | "GB";
  enabled: boolean;
  source?: string;
};

export type ThresholdRulePayload = {
  period_type: string;
  metric_type: string;
  threshold_value: number;
  threshold_unit: "MB" | "GB";
  enabled: boolean;
};

export const tabs: Array<{ key: TabKey; label: string; path: string }> = [
  { key: "overview", label: "总览", path: "/overview" },
  { key: "machines", label: "机器", path: "/machines" },
  { key: "sshKeys", label: "SSH Key", path: "/ssh-keys" },
  { key: "thresholds", label: "阈值", path: "/thresholds" },
  { key: "notifications", label: "通知", path: "/notifications" },
  { key: "samples", label: "样本", path: "/samples" },
  { key: "alerts", label: "告警", path: "/alerts" },
];

export const thresholdDimensions = [
  { period_type: "hourly", metric_type: "upload" },
  { period_type: "hourly", metric_type: "download" },
  { period_type: "hourly", metric_type: "total" },
  { period_type: "daily", metric_type: "upload" },
  { period_type: "daily", metric_type: "download" },
  { period_type: "daily", metric_type: "total" },
] as const;

export const emptyThresholdRows = (): ThresholdFormRow[] =>
  thresholdDimensions.map((dimension) => ({
    ...dimension,
    threshold_value: "",
    threshold_unit: "GB",
    enabled: false,
  }));

export function toThresholdPayloads(rows: ThresholdFormRow[]): ThresholdRulePayload[] {
  return rows.map((row) => ({
    period_type: row.period_type,
    metric_type: row.metric_type,
    threshold_value: Number(row.threshold_value || "0"),
    threshold_unit: row.threshold_unit,
    enabled: row.enabled,
  }));
}

export function toThresholdFormRows(
  rules: Array<{
    period_type: string;
    metric_type: string;
    threshold_value: number;
    threshold_unit: string;
    enabled: boolean;
    source?: string;
  }>,
): ThresholdFormRow[] {
  return thresholdDimensions.map((dimension) => {
    const matched = rules.find(
      (rule) => rule.period_type === dimension.period_type && rule.metric_type === dimension.metric_type,
    );

    return {
      ...dimension,
      threshold_value: matched ? String(matched.threshold_value) : "",
      threshold_unit: (matched?.threshold_unit as "MB" | "GB") || "GB",
      enabled: matched?.enabled ?? false,
      source: matched?.source,
    };
  });
}

export function safeParseHeaders(value: string): Record<string, string> {
  try {
    const parsed = JSON.parse(value) as Record<string, string>;
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch {
    return {};
  }
}

export function renderWebhookPreviewTemplate(value: string): string {
  return [
    ["{{machine_id}}", "1"],
    ["{{machine_name}}", "test-machine"],
    ["{{machine_host}}", "127.0.0.1"],
    ["{{period_type}}", "hourly"],
    ["{{metric_type}}", "total"],
    ["{{bucket_time}}", "2026-04-25 12:00:00 +0000 UTC"],
    ["{{bucket_time_rfc3339}}", "2026-04-25T12:00:00Z"],
    ["{{threshold_mb}}", "1024"],
    ["{{actual_mb}}", "1536"],
    ["{{alert_key}}", "test:webhook:alert"],
  ].reduce((result, [token, replacement]) => result.split(token).join(replacement), value);
}

export function renderWebhookPreviewHeaders(headers: Record<string, string>): Record<string, string> {
  return Object.fromEntries(
    Object.entries(headers).map(([key, value]) => [key, renderWebhookPreviewTemplate(value)]),
  );
}

export function toErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  return "unknown error";
}

export function formatTime(value: string) {
  return new Date(value).toLocaleString("zh-CN", { hour12: false });
}

export function formatNumber(value: number) {
  return value.toFixed(3);
}

export function formatTrafficValue(valueMB: number) {
  if (valueMB >= 1024 * 1024) {
    return `${(valueMB / (1024 * 1024)).toFixed(3)} TB`;
  }

  if (valueMB >= 1024) {
    return `${(valueMB / 1024).toFixed(3)} GB`;
  }

  return `${valueMB.toFixed(3)} MB`;
}

export function formatAlertPeriod(periodType: string, bucketTime: string) {
  const start = new Date(bucketTime);

  if (Number.isNaN(start.getTime())) {
    return bucketTime;
  }

  if (periodType === "hourly") {
    const end = new Date(start.getTime() + 60 * 60 * 1000 - 1000);
    return `${formatTime(bucketTime)} - ${end.toLocaleString("zh-CN", { hour12: false })}`;
  }

  if (periodType === "daily") {
    return `${start.toLocaleDateString("zh-CN")} 全天`;
  }

  return formatTime(bucketTime);
}

export function tabTitle(activeTab: TabKey) {
  return tabs.find((tab) => tab.key === activeTab)?.label ?? "管理台";
}

export function tabPath(tab: TabKey) {
  return tabs.find((item) => item.key === tab)?.path ?? "/overview";
}

export function tabKeyFromPath(pathname: string): TabKey {
  return tabs.find((tab) => tab.path === pathname)?.key ?? "overview";
}
