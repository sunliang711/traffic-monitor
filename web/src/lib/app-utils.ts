import type { Language } from "./i18n";
import { translate } from "./i18n";

export type TabKey =
  | "overview"
  | "machines"
  | "sshKeys"
  | "thresholds"
  | "notifications"
  | "samples"
  | "alerts";

export type MachineLabelOption = {
  value: number;
  label: string;
};

export type MachineDisplay = {
  primary: string;
  secondary?: string;
};

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

export const tabs: Array<{ key: TabKey; path: string }> = [
  { key: "overview", path: "/overview" },
  { key: "machines", path: "/machines" },
  { key: "sshKeys", path: "/ssh-keys" },
  { key: "samples", path: "/samples" },
  { key: "thresholds", path: "/thresholds" },
  { key: "notifications", path: "/notifications" },
  { key: "alerts", path: "/alerts" },
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

export function toThresholdPayloads(rows: ThresholdFormRow[], language: Language = "zh"): ThresholdRulePayload[] {
  return rows.map((row) => ({
    period_type: row.period_type,
    metric_type: row.metric_type,
    threshold_value: parseThresholdValue(row, language),
    threshold_unit: row.threshold_unit,
    enabled: row.enabled,
  }));
}

function parseThresholdValue(row: ThresholdFormRow, language: Language) {
  const rawValue = row.threshold_value.trim();
  const value = Number(rawValue || "0");

  if ((row.enabled || rawValue !== "") && (!Number.isFinite(value) || value <= 0)) {
    throw new Error(translate(language, "thresholdInvalidValue"));
  }

  return value;
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

export function safeParseHeaders(value: string, language: Language = "zh"): Record<string, string> {
  try {
    const parsed = JSON.parse(value) as Record<string, string>;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      throw new Error(translate(language, "notificationsHeadersInvalidObject"));
    }

    Object.entries(parsed).forEach(([key, headerValue]) => {
      if (typeof headerValue !== "string") {
        throw new Error(translate(language, "notificationsHeadersInvalidValue", { key }));
      }
    });

    return parsed;
  } catch (error) {
    if (error instanceof SyntaxError) {
      throw new Error(translate(language, "notificationsHeadersInvalidJson"));
    }

    if (error instanceof Error && error.message) {
      throw error;
    }

    throw new Error(translate(language, "notificationsHeadersInvalidJson"));
  }
}

export function tryParseHeaders(value: string): Record<string, string> {
  try {
    return safeParseHeaders(value);
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
    ["{{threshold_human_readable}}", "1.000 GB"],
    ["{{actual_mb}}", "1536"],
    ["{{actual_human_readable}}", "1.500 GB"],
    ["{{alert_key}}", "test:webhook:alert"],
  ].reduce((result, [token, replacement]) => result.split(token).join(replacement), value);
}

export function renderWebhookPreviewHeaders(headers: Record<string, string>): Record<string, string> {
  return Object.fromEntries(
    Object.entries(headers).map(([key, value]) => [key, renderWebhookPreviewTemplate(value)]),
  );
}

export function toErrorMessage(error: unknown, language: Language): string {
  if (error instanceof Error) {
    return error.message;
  }
  return translate(language, "unknownError");
}

function localeTag(language: Language) {
  return language === "zh" ? "zh-CN" : "en-US";
}

export function formatTime(value: string, language: Language) {
  return new Date(value).toLocaleString(localeTag(language), { hour12: false });
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

export function formatPeriodType(periodType: string, language: Language) {
  switch (periodType) {
    case "hourly":
      return translate(language, "periodHourly");
    case "daily":
      return translate(language, "periodDaily");
    default:
      return periodType;
  }
}

export function formatMetricType(metricType: string, language: Language) {
  switch (metricType) {
    case "upload":
      return translate(language, "metricUpload");
    case "download":
      return translate(language, "metricDownload");
    case "total":
      return translate(language, "metricTotal");
    default:
      return metricType;
  }
}

export function formatAlertPeriod(periodType: string, bucketTime: string, language: Language) {
  const start = new Date(bucketTime);

  if (Number.isNaN(start.getTime())) {
    return bucketTime;
  }

  if (periodType === "hourly") {
    const end = new Date(start.getTime() + 60 * 60 * 1000 - 1000);
    return `${formatTime(bucketTime, language)} - ${end.toLocaleString(localeTag(language), { hour12: false })}`;
  }

  if (periodType === "daily") {
    return `${start.toLocaleDateString(localeTag(language))} ${translate(language, "allDay")}`;
  }

  return formatTime(bucketTime, language);
}

export function tabLabel(activeTab: TabKey, language: Language) {
  switch (activeTab) {
    case "overview":
      return translate(language, "tabOverview");
    case "machines":
      return translate(language, "tabMachines");
    case "sshKeys":
      return translate(language, "tabSSHKeys");
    case "thresholds":
      return translate(language, "tabThresholds");
    case "notifications":
      return translate(language, "tabNotifications");
    case "samples":
      return translate(language, "tabSamples");
    case "alerts":
      return translate(language, "tabAlerts");
    default:
      return translate(language, "consoleTitle");
  }
}

export function tabTitle(activeTab: TabKey, language: Language) {
  return tabLabel(activeTab, language);
}

export function tabPath(tab: TabKey) {
  return tabs.find((item) => item.key === tab)?.path ?? "/overview";
}

export function tabKeyFromPath(pathname: string): TabKey {
  return tabs.find((tab) => tab.path === pathname)?.key ?? "overview";
}

export function machineLabel(machineOptions: MachineLabelOption[], machineID: number, language: Language) {
  return machineOptions.find((option) => option.value === machineID)?.label ?? translate(language, "machineLabel", { id: machineID });
}

export function machineDisplay(machineOptions: MachineLabelOption[], machineID: number, language: Language): MachineDisplay {
  const label = machineLabel(machineOptions, machineID, language);
  const matched = label.match(/^(.*)\s+\((.*)\)$/);

  if (!matched) {
    return { primary: label };
  }

  return {
    primary: matched[1],
    secondary: matched[2],
  };
}

export function formatStatusText(status: string, language: Language) {
  switch (status.toLowerCase()) {
    case "success":
      return translate(language, "statusSuccess");
    case "sent":
      return translate(language, "statusSent");
    case "ok":
      return translate(language, "statusOk");
    case "pending":
      return translate(language, "statusPending");
    case "queued":
      return translate(language, "statusQueued");
    case "processing":
      return translate(language, "statusProcessing");
    case "failed":
      return translate(language, "statusFailed");
    case "error":
      return translate(language, "statusError");
    default:
      return status;
  }
}

export function formatThresholdSource(source: string | undefined, language: Language) {
  if (!source) {
    return translate(language, "sourceUnknown");
  }

  switch (source.toLowerCase()) {
    case "global":
      return translate(language, "sourceGlobal");
    case "machine":
      return translate(language, "sourceMachine");
    case "inherited":
      return translate(language, "sourceInherited");
    case "default":
      return translate(language, "sourceDefault");
    default:
      return source;
  }
}
