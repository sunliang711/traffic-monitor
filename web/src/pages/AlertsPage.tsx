import type { AlertItem } from "../types";
import type { MachineOption } from "../lib/app-types";
import {
  formatAlertPeriod,
  formatMetricType,
  formatPeriodType,
  formatStatusText,
  formatTime,
  formatTrafficValue,
  machineDisplay,
} from "../lib/app-utils";
import { useI18n } from "../lib/i18n";
import EmptyState from "../components/EmptyState";
import Pagination from "../components/Pagination";

type AlertsPageProps = {
  busy: boolean;
  alerts: AlertItem[];
  total: number;
  page: number;
  pageSize: number;
  machineOptions: MachineOption[];
  selectedMachineID: number | null;
  onSelectMachine: (machineID: number | null) => void | Promise<void>;
  onPageChange: (page: number) => void | Promise<void>;
  onPageSizeChange: (pageSize: number) => void | Promise<void>;
};

function notifyStatusBadgeClass(status: string) {
  switch (status.toLowerCase()) {
    case "success":
    case "sent":
    case "ok":
      return "ok";
    case "pending":
    case "queued":
    case "processing":
      return "idle";
    case "failed":
    case "error":
      return "error";
    default:
      return "idle";
  }
}

export default function AlertsPage(props: AlertsPageProps) {
  const { language, t } = useI18n();
  const totalPages = Math.max(1, Math.ceil(props.total / props.pageSize));

  return (
    <div className="page-stack">
      <section className="summary-strip">
        <div className="summary-tile amber compact">
          <span>{t("overviewAlertsLabel")}</span>
          <strong>{props.total}</strong>
        </div>
      </section>

      <section className="panel section-panel">
        <div className="section-intro">
          <div>
            <h3 className="panel-title">{t("alertsTitle")}</h3>
          </div>
          <p className="section-description">{t("alertsPageDescription")}</p>
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
        </div>

        {props.alerts.length === 0 ? (
          <EmptyState title={t("alertsEmptyTitle")} description={t("alertsEmptyDescription")} />
        ) : (
          <>
            <div className="table-wrapper responsive-table">
              <table>
                <thead>
                  <tr>
                    <th>{t("tabMachines")}</th>
                    <th>{t("thresholdPeriod")}</th>
                    <th>{t("thresholdDimension")}</th>
                    <th>{t("alertsAlertWindow")}</th>
                    <th>{t("alertsThreshold")}</th>
                    <th>{t("alertsActual")}</th>
                    <th>{t("alertsNotifyStatus")}</th>
                    <th>{t("alertsNotifyTime")}</th>
                  </tr>
                </thead>
                <tbody>
                  {props.alerts.map((alert) => {
                    const machine = machineDisplay(props.machineOptions, alert.machine_id, language);

                    return (
                      <tr key={alert.id}>
                        <td>
                          <div className="machine-cell">
                            <strong>{machine.primary}</strong>
                            {machine.secondary ? <span className="machine-host">{machine.secondary}</span> : null}
                          </div>
                        </td>
                        <td>{formatPeriodType(alert.period_type, language)}</td>
                        <td>{formatMetricType(alert.metric_type, language)}</td>
                        <td>{formatAlertPeriod(alert.period_type, alert.bucket_time, language)}</td>
                        <td>{formatTrafficValue(alert.threshold_mb)}</td>
                        <td>{formatTrafficValue(alert.actual_mb)}</td>
                        <td>
                          <span className={`status-badge ${notifyStatusBadgeClass(alert.notify_status)}`}>
                            {formatStatusText(alert.notify_status, language)}
                          </span>
                        </td>
                        <td>{alert.notified_at ? formatTime(alert.notified_at, language) : "-"}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
            <div className="mobile-card-list">
              {props.alerts.map((alert) => {
                const machine = machineDisplay(props.machineOptions, alert.machine_id, language);

                return (
                  <article className="card mobile-record-card" key={alert.id}>
                    <div className="card-header">
                      <div className="machine-cell">
                        <strong>{machine.primary}</strong>
                        {machine.secondary ? <span className="machine-host">{machine.secondary}</span> : null}
                      </div>
                      <span className={`status-badge ${notifyStatusBadgeClass(alert.notify_status)}`}>
                        {formatStatusText(alert.notify_status, language)}
                      </span>
                    </div>
                    <dl className="record-grid">
                      <div>
                        <dt>{t("thresholdPeriod")}</dt>
                        <dd>{formatPeriodType(alert.period_type, language)}</dd>
                      </div>
                      <div>
                        <dt>{t("thresholdDimension")}</dt>
                        <dd>{formatMetricType(alert.metric_type, language)}</dd>
                      </div>
                      <div>
                        <dt>{t("alertsAlertWindow")}</dt>
                        <dd>{formatAlertPeriod(alert.period_type, alert.bucket_time, language)}</dd>
                      </div>
                      <div>
                        <dt>{t("alertsThreshold")}</dt>
                        <dd>{formatTrafficValue(alert.threshold_mb)}</dd>
                      </div>
                      <div>
                        <dt>{t("alertsActual")}</dt>
                        <dd>{formatTrafficValue(alert.actual_mb)}</dd>
                      </div>
                      <div>
                        <dt>{t("alertsNotifyTime")}</dt>
                        <dd>{alert.notified_at ? formatTime(alert.notified_at, language) : "-"}</dd>
                      </div>
                    </dl>
                  </article>
                );
              })}
            </div>
            <Pagination
              page={props.page}
              totalPages={totalPages}
              total={props.total}
              pageSize={props.pageSize}
              disabled={props.busy}
              onPageChange={(page) => void props.onPageChange(page)}
              onPageSizeChange={(pageSize) => void props.onPageSizeChange(pageSize)}
            />
          </>
        )}
      </section>
    </div>
  );
}
