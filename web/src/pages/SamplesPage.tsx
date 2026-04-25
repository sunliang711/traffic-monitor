import type { CollectNowResponse, TrafficSample } from "../types";
import type { MachineOption } from "../lib/app-types";
import { formatPeriodType, formatStatusText, formatTime, formatTrafficValue, machineDisplay } from "../lib/app-utils";
import { useI18n } from "../lib/i18n";
import EmptyState from "../components/EmptyState";

type SamplesPageProps = {
  busy: boolean;
  selectedMachineID: number | null;
  machineOptions: MachineOption[];
  samples: TrafficSample[];
  total: number;
  page: number;
  pageSize: number;
  collectResults: CollectNowResponse["results"];
  onSelectMachine: (machineID: number | null) => void | Promise<void>;
  onPageChange: (page: number) => void | Promise<void>;
  onCollectCurrentMachine: (machineID: number) => void;
};

export default function SamplesPage(props: SamplesPageProps) {
  const { language, t } = useI18n();
  const totalPages = Math.max(1, Math.ceil(props.total / props.pageSize));

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
        <div className="section-intro">
          <div>
            <p className="section-kicker">{t("samplesTitle")}</p>
            <h3 className="panel-title">{t("samplesTitle")}</h3>
          </div>
          <p className="section-description">{t("samplesPageDescription")}</p>
        </div>
        <div className="toolbar-row">
          <div className="toolbar-filters">
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
          </div>
          {props.selectedMachineID ? (
            <button
              className="secondary-button"
              onClick={() => props.onCollectCurrentMachine(props.selectedMachineID!)}
              type="button"
            >
              {t("samplesCollectCurrentMachine")}
            </button>
          ) : null}
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

                    return (
                      <tr key={`${sample.id}-${sample.period_type}`}>
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

                return (
                  <article className="card mobile-record-card" key={`${sample.id}-${sample.period_type}`}>
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
              <span className="card-meta">{t("samplesPageInfo", { page: props.page, totalPages, total: props.total })}</span>
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
            <p className="section-kicker">{t("samplesRecentResults")}</p>
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
