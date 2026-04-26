import { useEffect, useRef, useState } from "react";
import type { CollectNowResponse, TrafficSample } from "../types";
import type { MachineOption } from "../lib/app-types";
import {
  bucketGroupToneClassName,
  buildBucketToneMap,
  formatPeriodType,
  formatStatusText,
  formatTime,
  formatTrafficValue,
  machineDisplay,
} from "../lib/app-utils";
import { useI18n } from "../lib/i18n";
import EmptyState from "../components/EmptyState";
import PageSizeSelect from "../components/PageSizeSelect";

const autoRefreshOptions = [5, 10, 15, 30];
const sampleAutoRefreshStorageKey = "traffic-monitor-samples-auto-refresh";

type SampleAutoRefreshPreference = {
  enabled: boolean;
  interval: number;
};

const defaultAutoRefreshPreference: SampleAutoRefreshPreference = {
  enabled: false,
  interval: 30,
};

function readSampleAutoRefreshPreference(): SampleAutoRefreshPreference {
  if (typeof window === "undefined") {
    return defaultAutoRefreshPreference;
  }

  try {
    const raw = window.localStorage.getItem(sampleAutoRefreshStorageKey);
    if (!raw) {
      return defaultAutoRefreshPreference;
    }

    const parsed = JSON.parse(raw) as Partial<SampleAutoRefreshPreference>;
    return {
      enabled: parsed.enabled === true,
      interval: typeof parsed.interval === "number" && autoRefreshOptions.includes(parsed.interval)
        ? parsed.interval
        : defaultAutoRefreshPreference.interval,
    };
  } catch {
    return defaultAutoRefreshPreference;
  }
}

type SamplesPageProps = {
  busy: boolean;
  selectedMachineID: number | null;
  selectedPeriodType: string;
  machineOptions: MachineOption[];
  samples: TrafficSample[];
  total: number;
  page: number;
  pageSize: number;
  collectResults: CollectNowResponse["results"];
  onSelectMachine: (machineID: number | null) => void | Promise<void>;
  onSelectPeriodType: (periodType: string) => void | Promise<void>;
  onPageChange: (page: number) => void | Promise<void>;
  onPageSizeChange: (pageSize: number) => void | Promise<void>;
  onRefresh: () => void | Promise<void>;
  onCollectAllMachines: () => void;
  onCollectCurrentMachine: (machineID: number) => void;
};

function RefreshIcon() {
  return (
    <svg className="refresh-icon" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path d="M20 6v5h-5" />
      <path d="M4 18v-5h5" />
      <path d="M17.7 9A6.5 6.5 0 0 0 6.2 6.8L4 9" />
      <path d="M6.3 15A6.5 6.5 0 0 0 17.8 17.2L20 15" />
    </svg>
  );
}

export default function SamplesPage(props: SamplesPageProps) {
  const { language, t } = useI18n();
  const refreshCallback = useRef(props.onRefresh);
  const busyRef = useRef(props.busy);
  const autoRefreshWrapperRef = useRef<HTMLDivElement | null>(null);
  const [isAutoRefreshMenuOpen, setAutoRefreshMenuOpen] = useState(false);
  const [autoRefreshPreference, setAutoRefreshPreference] = useState(readSampleAutoRefreshPreference);
  const autoRefreshEnabled = autoRefreshPreference.enabled;
  const autoRefreshInterval = autoRefreshPreference.interval;
  const sampleBucketToneMap = buildBucketToneMap(props.samples);
  const totalPages = Math.max(1, Math.ceil(props.total / props.pageSize));

  useEffect(() => {
    refreshCallback.current = props.onRefresh;
  }, [props.onRefresh]);

  useEffect(() => {
    busyRef.current = props.busy;
  }, [props.busy]);

  useEffect(() => {
    if (!isAutoRefreshMenuOpen) {
      return;
    }

    const handlePointerDown = (event: PointerEvent) => {
      if (event.target instanceof Node && autoRefreshWrapperRef.current?.contains(event.target)) {
        return;
      }

      setAutoRefreshMenuOpen(false);
    };

    document.addEventListener("pointerdown", handlePointerDown);
    return () => document.removeEventListener("pointerdown", handlePointerDown);
  }, [isAutoRefreshMenuOpen]);

  useEffect(() => {
    try {
      window.localStorage.setItem(sampleAutoRefreshStorageKey, JSON.stringify(autoRefreshPreference));
    } catch {
      // 忽略本地存储写入失败，自动刷新控件仍按内存状态工作。
    }
  }, [autoRefreshPreference]);

  useEffect(() => {
    if (!autoRefreshEnabled) {
      return;
    }

    const timer = window.setInterval(() => {
      if (!busyRef.current) {
        void refreshCallback.current();
      }
    }, autoRefreshInterval * 1000);

    return () => window.clearInterval(timer);
  }, [autoRefreshEnabled, autoRefreshInterval]);

  return (
    <div className="page-stack">
      <section className="summary-strip">
        <div className="summary-tile teal compact">
          <span>{t("overviewSamplesLabel")}</span>
          <strong>{props.total}</strong>
        </div>
        <div className="summary-tile amber compact">
          <span>{t("overviewCollectLabel")}</span>
          <strong>{props.collectResults.length > 0 ? formatStatusText(props.collectResults[0].status, language) : t("statusNotRun")}</strong>
        </div>
      </section>

      <section className="panel section-panel">
        <div className="section-toolbar samples-section-toolbar">
          <div className="section-intro">
            <div>
              <h3 className="panel-title">{t("samplesTitle")}</h3>
            </div>
            <p className="section-description">{t("samplesPageDescription")}</p>
          </div>
          <div className="sample-refresh-actions">
            <button
              className="secondary-button sample-refresh-button"
              disabled={props.busy}
              onClick={() => void props.onRefresh()}
              title={t("refresh")}
              type="button"
              aria-label={t("refresh")}
            >
              <RefreshIcon />
            </button>
            <div className="sample-auto-refresh-wrapper" ref={autoRefreshWrapperRef}>
              <button
                className={`secondary-button sample-auto-refresh-button${isAutoRefreshMenuOpen ? " open" : ""}${autoRefreshEnabled ? " enabled" : ""}`}
                aria-expanded={isAutoRefreshMenuOpen}
                aria-haspopup="menu"
                onClick={() => setAutoRefreshMenuOpen((current) => !current)}
                type="button"
              >
                <RefreshIcon />
                <span className="sample-auto-refresh-label">{t("samplesAutoRefresh")}</span>
                <span className={`sample-auto-refresh-state${autoRefreshEnabled ? " enabled" : ""}`}>
                  {autoRefreshEnabled
                    ? t("samplesAutoRefreshOn", { seconds: autoRefreshInterval })
                    : t("samplesAutoRefreshOff")}
                </span>
              </button>
              {isAutoRefreshMenuOpen ? (
                <div className="sample-auto-refresh-menu" role="menu">
                  <button
                    className={`sample-auto-refresh-menu-item${autoRefreshEnabled ? " active" : ""}`}
                    onClick={() => setAutoRefreshPreference((current) => ({ ...current, enabled: !current.enabled }))}
                    role="menuitemcheckbox"
                    aria-checked={autoRefreshEnabled}
                    type="button"
                  >
                    <span>{t("samplesEnableAutoRefresh")}</span>
                    {autoRefreshEnabled ? <span className="sample-auto-refresh-check" aria-hidden="true" /> : null}
                  </button>
                  <div className="sample-auto-refresh-divider" />
                  {autoRefreshOptions.map((seconds) => (
                    <button
                      className={`sample-auto-refresh-menu-item${autoRefreshInterval === seconds ? " active" : ""}`}
                      onClick={() => {
                        setAutoRefreshPreference({ enabled: true, interval: seconds });
                        setAutoRefreshMenuOpen(false);
                      }}
                      role="menuitemradio"
                      aria-checked={autoRefreshInterval === seconds}
                      type="button"
                      key={seconds}
                    >
                      <span>{t("samplesRefreshSeconds", { seconds })}</span>
                      {autoRefreshInterval === seconds ? <span className="sample-auto-refresh-check" aria-hidden="true" /> : null}
                    </button>
                  ))}
                </div>
              ) : null}
            </div>
          </div>
        </div>
        <div className="toolbar-row">
          <div className="toolbar-filters sample-toolbar-filters">
            <select
              value={props.selectedMachineID ?? ""}
              onChange={(event) => void props.onSelectMachine(event.target.value ? Number(event.target.value) : null)}
            >
              <option value="">{t("samplesAllMachines")}</option>
              {props.machineOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
            <select
              aria-label={t("thresholdPeriod")}
              value={props.selectedPeriodType}
              onChange={(event) => void props.onSelectPeriodType(event.target.value)}
            >
              <option value="">{t("samplesAllPeriods")}</option>
              <option value="hourly">{formatPeriodType("hourly", language)}</option>
              <option value="daily">{formatPeriodType("daily", language)}</option>
            </select>
          </div>
          <div className="action-row sample-collect-actions">
            <button
              className="secondary-button"
              disabled={props.busy}
              onClick={props.onCollectAllMachines}
              type="button"
            >
              {t("collectAllMachines")}
            </button>
            {props.selectedMachineID ? (
              <button
                className="secondary-button"
                disabled={props.busy}
                onClick={() => props.onCollectCurrentMachine(props.selectedMachineID!)}
                type="button"
              >
                {t("samplesCollectCurrentMachine")}
              </button>
            ) : null}
          </div>
        </div>

        {props.samples.length === 0 ? (
          <EmptyState title={t("samplesEmptyTitle")} description={t("samplesEmptyDescription")} />
        ) : (
          <>
            <div className="table-wrapper responsive-table">
              <table>
                <thead>
                  <tr>
                    <th>{t("tabMachines")}</th>
                    <th>{t("thresholdPeriod")}</th>
                    <th>{t("samplesBucketTime")}</th>
                    <th>{t("metricUpload")}</th>
                    <th>{t("metricDownload")}</th>
                    <th>{t("metricTotal")}</th>
                    <th>{t("samplesCollectedAt")}</th>
                  </tr>
                </thead>
                <tbody>
                  {props.samples.map((sample) => {
                    const machine = machineDisplay(props.machineOptions, sample.machine_id, language);
                    const bucketToneClassName = bucketGroupToneClassName(sample.period_type, sample.bucket_time, sampleBucketToneMap);

                    return (
                      <tr className={`bucket-group-row ${bucketToneClassName}`} key={`${sample.id}-${sample.period_type}`}>
                        <td>
                          <div className="machine-cell">
                            <strong>{machine.primary}</strong>
                            {machine.secondary ? <span className="machine-host">{machine.secondary}</span> : null}
                          </div>
                        </td>
                        <td>{formatPeriodType(sample.period_type, language)}</td>
                        <td>{formatTime(sample.bucket_time, language)}</td>
                        <td>{formatTrafficValue(sample.upload_mb)}</td>
                        <td>{formatTrafficValue(sample.download_mb)}</td>
                        <td>{formatTrafficValue(sample.total_mb)}</td>
                        <td>{formatTime(sample.collected_at, language)}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
            <div className="mobile-card-list">
              {props.samples.map((sample) => {
                const machine = machineDisplay(props.machineOptions, sample.machine_id, language);
                const bucketToneClassName = bucketGroupToneClassName(sample.period_type, sample.bucket_time, sampleBucketToneMap);

                return (
                  <article className={`card mobile-record-card bucket-group-card ${bucketToneClassName}`} key={`${sample.id}-${sample.period_type}`}>
                    <div className="card-header">
                      <div className="machine-cell">
                        <strong>{machine.primary}</strong>
                        {machine.secondary ? <span className="machine-host">{machine.secondary}</span> : null}
                      </div>
                      <span className="status-badge idle">{formatPeriodType(sample.period_type, language)}</span>
                    </div>
                    <dl className="record-grid">
                      <div>
                        <dt>{t("samplesBucketTime")}</dt>
                        <dd>{formatTime(sample.bucket_time, language)}</dd>
                      </div>
                      <div>
                        <dt>{t("metricUpload")}</dt>
                        <dd>{formatTrafficValue(sample.upload_mb)}</dd>
                      </div>
                      <div>
                        <dt>{t("metricDownload")}</dt>
                        <dd>{formatTrafficValue(sample.download_mb)}</dd>
                      </div>
                      <div>
                        <dt>{t("metricTotal")}</dt>
                        <dd>{formatTrafficValue(sample.total_mb)}</dd>
                      </div>
                      <div>
                        <dt>{t("samplesCollectedAt")}</dt>
                        <dd>{formatTime(sample.collected_at, language)}</dd>
                      </div>
                    </dl>
                  </article>
                );
              })}
            </div>
            <div className="pagination-row">
              <div className="pagination-meta">
                <span className="card-meta">{t("samplesPageInfo", { page: props.page, totalPages, total: props.total })}</span>
                <PageSizeSelect value={props.pageSize} onChange={(pageSize) => void props.onPageSizeChange(pageSize)} />
              </div>
              <div className="action-row">
                <button
                  className="secondary-button"
                  disabled={props.busy || props.page <= 1}
                  onClick={() => void props.onPageChange(props.page - 1)}
                  type="button"
                >
                  {t("previousPage")}
                </button>
                <button
                  className="secondary-button"
                  disabled={props.busy || props.page >= totalPages}
                  onClick={() => void props.onPageChange(props.page + 1)}
                  type="button"
                >
                  {t("nextPage")}
                </button>
              </div>
            </div>
          </>
        )}
      </section>

      <section className="panel section-panel">
        <div className="section-intro">
          <div>
            <h3 className="panel-title">{t("samplesRecentResults")}</h3>
          </div>
        </div>
        {props.collectResults.length === 0 ? (
          <EmptyState title={t("samplesResultsEmptyTitle")} description={t("samplesResultsEmptyDescription")} />
        ) : (
          <div className="list-block card-list result-list">
            {props.collectResults.map((result) => {
              const machine = machineDisplay(props.machineOptions, result.machine_id, language);

              return (
                <article className="card result-card" key={`${result.machine_id}-${result.status}`}>
                  <div className="card-header">
                    <div className="machine-cell">
                      <strong>{machine.primary}</strong>
                      {machine.secondary ? <span className="machine-host">{machine.secondary}</span> : null}
                    </div>
                    <span className={`status-badge ${result.status === "success" ? "ok" : "error"}`}>
                      {formatStatusText(result.status, language)}
                    </span>
                  </div>
                  <p className="card-meta">{t("samplesSampleCount", { count: result.sample_count })}</p>
                  {result.error ? <p className="card-meta">{t("samplesError", { error: result.error })}</p> : null}
                </article>
              );
            })}
          </div>
        )}
      </section>
    </div>
  );
}
