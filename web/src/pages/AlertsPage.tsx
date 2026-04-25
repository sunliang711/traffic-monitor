import type { AlertItem } from "../types";
import type { MachineOption } from "../lib/app-types";
import {
  formatAlertPeriod,
  formatMetricType,
  formatPeriodType,
  formatTime,
  formatTrafficValue,
  machineDisplay,
} from "../lib/app-utils";

type AlertsPageProps = {
  alerts: AlertItem[];
  machineOptions: MachineOption[];
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
  return (
    <section className="panel">
      <h3 className="panel-title">告警记录</h3>
      <div className="table-wrapper">
        <table>
          <thead>
            <tr>
              <th>机器</th>
              <th>周期</th>
              <th>维度</th>
              <th>告警周期</th>
              <th>阈值</th>
              <th>实际</th>
              <th>通知状态</th>
              <th>通知时间</th>
            </tr>
          </thead>
          <tbody>
            {props.alerts.map((alert) => {
              const machine = machineDisplay(props.machineOptions, alert.machine_id);

              return (
                <tr key={alert.id}>
                  <td>
                    <div className="machine-cell">
                      <strong>{machine.primary}</strong>
                      {machine.secondary ? <span className="machine-host">{machine.secondary}</span> : null}
                    </div>
                  </td>
                  <td>{formatPeriodType(alert.period_type)}</td>
                  <td>{formatMetricType(alert.metric_type)}</td>
                  <td>{formatAlertPeriod(alert.period_type, alert.bucket_time)}</td>
                  <td>{formatTrafficValue(alert.threshold_mb)}</td>
                  <td>{formatTrafficValue(alert.actual_mb)}</td>
                  <td>
                    <span className={`status-badge ${notifyStatusBadgeClass(alert.notify_status)}`}>
                      {alert.notify_status}
                    </span>
                  </td>
                  <td>{alert.notified_at ? formatTime(alert.notified_at) : "-"}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </section>
  );
}
