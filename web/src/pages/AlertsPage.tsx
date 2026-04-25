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

type AlertsPageProps = {
  alerts: AlertItem[];
  machineOptions: MachineOption[];
  selectedMachineID: number | null;
  onSelectMachine: (machineID: number | null) => void;
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
  const filteredAlerts = props.alerts.filter(
    (alert) => !props.selectedMachineID || alert.machine_id === props.selectedMachineID,
  );

  return (
    <section className="panel">
      <div className="panel-header-inline">
        <h3 className="panel-title">{t("alertsTitle")}</h3>
        <select
          value={props.selectedMachineID ?? ""}
          onChange={(event) => props.onSelectMachine(event.target.value ? Number(event.target.value) : null)}
        >
          <option value="">{t("samplesAllMachines")}</option>
          {props.machineOptions.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
      </div>
      <div className="table-wrapper">
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
            {filteredAlerts.map((alert) => {
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
    </section>
  );
}
