import type { AlertItem } from "../types";
import type { MachineOption } from "../lib/app-types";
import { formatAlertPeriod, formatTime, formatTrafficValue, machineLabel } from "../lib/app-utils";

type AlertsPageProps = {
  alerts: AlertItem[];
  machineOptions: MachineOption[];
};

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
            {props.alerts.map((alert) => (
              <tr key={alert.id}>
                <td>{machineLabel(props.machineOptions, alert.machine_id)}</td>
                <td>{alert.period_type}</td>
                <td>{alert.metric_type}</td>
                <td>{formatAlertPeriod(alert.period_type, alert.bucket_time)}</td>
                <td>{formatTrafficValue(alert.threshold_mb)}</td>
                <td>{formatTrafficValue(alert.actual_mb)}</td>
                <td>{alert.notify_status}</td>
                <td>{alert.notified_at ? formatTime(alert.notified_at) : "-"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
